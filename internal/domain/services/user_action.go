package services

import (
	"crypto/subtle"
	stdErr "errors"
	"fmt"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"log/slog"
	"time"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/models"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"gorm.io/gorm"
)

type UserActionCreateIn struct {
	Login  string
	Realm  string
	Action string
	Data   *string
	// TTL: the action expires that long after its creation; 0: never
	TTL time.Duration
	// MaxAttempts: the wrong codes Validate takes with Login + Action; 0: no limit
	MaxAttempts int
}

type UserActionCreateOut struct {
	Data string
}

func UserActionCreate(
	db *gorm.DB,
	in UserActionCreateIn,
	uuidGen func() string,
	timeFn func() time.Time,
) (UserActionCreateOut, error) {
	now := timeFn()
	if in.TTL < 0 || in.MaxAttempts < 0 {
		return UserActionCreateOut{}, errors.UnprocessableEntity(stdErr.New(consts.ERR_INVALID_INPUT_USER_ACTION_LIMITS))
	}
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "realm_name", in.Realm)
		return UserActionCreateOut{}, errors.BadRequest(err)
	}
	if err := KindOf(realm).AssertPasswordFlows(); err != nil {
		return UserActionCreateOut{}, err
	}
	user := entities.User{}
	if err := db.First(&user, "login = ? AND realm_id = ?", in.Login, realm.ID).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "login", in.Login, "realm_name", in.Realm)
		return UserActionCreateOut{}, errors.BadRequest(err)
	}
	action := entities.UserAction{}
	actionRes := db.First(&action, "user_id = ? AND realm_id = ? AND action = ? AND validated_at IS NULL", user.ID, realm.ID, in.Action)
	if actionRes.Error == nil && action.ID != 0 {
		if err := db.Model(&action).Update("validated_at", now).Error; err != nil {
			return UserActionCreateOut{}, errors.BadRequest(fmt.Errorf("UserActionCreate: %w", err))
		}
	}
	data := dt.Deref(in.Data)
	if in.Action == entities.UserActionTypePassword || data == "" {
		data = uuidGen()
	}
	dom := entities.UserAction{
		UserID:      user.ID,
		RealmID:     realm.ID,
		Action:      in.Action,
		Data:        data,
		ExpiresAt:   expiresAt(now, in.TTL),
		MaxAttempts: in.MaxAttempts,
	}
	if res := db.Create(&dom); res.Error != nil {
		return UserActionCreateOut{}, errors.DBError(res.Error)
	}
	return UserActionCreateOut{
		Data: data,
	}, nil
}

// expiresAt is when an action made at now with this TTL expires; nil: never.
func expiresAt(now time.Time, ttl time.Duration) *time.Time {
	if ttl == 0 {
		return nil
	}
	return dt.Ptr(now.Add(ttl))
}

type UserActionValidateIn struct {
	Login             string
	Realm             string
	Data              string
	Against           string
	SelfContainedCode bool
	// Action: with Login, Data is checked against the user's pending action of this type,
	// so a wrong code counts as a try. Empty: the action is the one whose Data matches.
	Action string
}

// byAction tells if Data is checked against the user's pending action of type Action.
func (in UserActionValidateIn) byAction() bool {
	return !in.SelfContainedCode && in.Action != ""
}

// countsTries tells if the try limit applies: not to a self-contained code, which has no
// login to count against (and a UUID can't be guessed).
func (in UserActionValidateIn) countsTries() bool {
	return !in.SelfContainedCode
}

func getUserAndAction(
	db *gorm.DB,
	realm entities.Realm,
	in UserActionValidateIn,
) (*entities.User, *entities.UserAction, error) {
	if in.SelfContainedCode {
		action := entities.UserAction{}
		actionRes := db.First(&action, "realm_id = ? AND data = ? AND validated_at IS NULL", realm.ID, in.Data)
		if actionRes.Error != nil {
			return nil, nil, errors.NotFound(actionRes.Error)
		}
		user := entities.User{}
		if err := db.First(&user, "id = ?", action.UserID).Error; err != nil {
			slog.ErrorContext(logs.DBContext(db), err.Error(), "user_id", action.UserID)
			return nil, nil, errors.BadRequest(err)
		}
		return &user, &action, nil
	}
	user := entities.User{}
	if err := db.First(&user, "login = ? AND realm_id = ?", in.Login, realm.ID).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "login", in.Login, "realm_name", in.Realm)
		return nil, nil, errors.BadRequest(err)
	}
	action := entities.UserAction{}
	var actionRes *gorm.DB
	if in.byAction() {
		actionRes = db.Last(&action, "user_id = ? AND realm_id = ? AND action = ? AND validated_at IS NULL", user.ID, realm.ID, in.Action)
	} else {
		actionRes = db.First(&action, "user_id = ? AND realm_id = ? AND data = ? AND validated_at IS NULL", user.ID, realm.ID, in.Data)
	}
	if actionRes.Error != nil {
		return nil, nil, errors.NotFound(actionRes.Error)
	}
	return &user, &action, nil
}

func UserActionValidate(
	db *gorm.DB,
	usersParams *models.UsersParams,
	in UserActionValidateIn,
	timeFn func() time.Time,
) (string, error) {
	now := timeFn()
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "realm_name", in.Realm)
		return "", errors.BadRequest(err)
	}
	if err := KindOf(realm).AssertPasswordFlows(); err != nil {
		return "", err
	}
	user, action, err := getUserAndAction(db, realm, in)
	if err != nil {
		return "", err
	}
	if user == nil || action == nil {
		return "", errors.InternalServerError(fmt.Errorf("user or action not found"))
	}
	if err := usable(*action, now, in.countsTries()); err != nil {
		return "", err
	}
	if in.byAction() && subtle.ConstantTimeCompare([]byte(action.Data), []byte(in.Data)) != 1 {
		return "", useATry(db, *action)
	}
	if err := validateAction(db, usersParams, *user, *action, in, now); err != nil {
		return "", err
	}
	return user.Login, nil
}

// usable is nil when the action can still be validated at now, else 410: it expired, or
// (when countsTries) its tries are used up.
func usable(action entities.UserAction, now time.Time, countsTries bool) error {
	if action.Expired(now) {
		return errors.Gone(stdErr.New(consts.ERR_USER_ACTION_EXPIRED))
	}
	if countsTries && action.TriesUsedUp() {
		return errors.Gone(stdErr.New(consts.ERR_USER_ACTION_TOO_MANY_ATTEMPTS))
	}
	return nil
}

// useATry counts a wrong code in one statement, so wrong codes sent at once never use more
// tries than the action allows: 404 as for any wrong code, or 410 when no try was left.
func useATry(db *gorm.DB, action entities.UserAction) error {
	res := db.Model(&entities.UserAction{}).
		Where("id = ? AND (max_attempts = 0 OR attempts < max_attempts)", action.ID).
		UpdateColumn("attempts", gorm.Expr("attempts + 1"))
	if res.Error != nil {
		return errors.DBError(fmt.Errorf("useATry: %w", res.Error))
	}
	if res.RowsAffected == 0 {
		return errors.Gone(stdErr.New(consts.ERR_USER_ACTION_TOO_MANY_ATTEMPTS))
	}
	return errors.NotFound(gorm.ErrRecordNotFound)
}

// validateAction marks the action validated and does what it is for, in one transaction:
// either both happen, or neither and the action stays pending.
func validateAction(
	db *gorm.DB,
	usersParams *models.UsersParams,
	user entities.User,
	action entities.UserAction,
	in UserActionValidateIn,
	now time.Time,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		claimed, err := claimAction(tx, action, now, in.countsTries())
		if err != nil {
			return err
		}
		if !claimed {
			return refusal(tx, action.ID, now, in.countsTries())
		}
		if action.Action != entities.UserActionTypePassword {
			return nil
		}
		if err := userActionResetPassword(tx, usersParams, user, in.Against, now); err != nil {
			return errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
		}
		return nil
	})
}

// claimAction marks the action validated, only while it is pending, unexpired and (when
// countsTries) with tries left: a parallel validation or wrong code can't slip past usable.
func claimAction(tx *gorm.DB, action entities.UserAction, now time.Time, countsTries bool) (bool, error) {
	query := tx.Model(&entities.UserAction{}).
		Where("id = ? AND validated_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", action.ID, now)
	if countsTries {
		query = query.Where("(max_attempts = 0 OR attempts < max_attempts)")
	}
	res := query.UpdateColumns(map[string]any{"validated_at": now, "updated_at": now})
	if res.Error != nil {
		return false, errors.BadRequest(fmt.Errorf("UserActionValidate: %w", res.Error))
	}
	return res.RowsAffected == 1, nil
}

// refusal tells why an action could not be claimed: 410 when it expired or used its tries up
// meanwhile, else 404, since a parallel call validated it.
func refusal(tx *gorm.DB, actionID uint, now time.Time, countsTries bool) error {
	var action entities.UserAction
	if err := tx.First(&action, actionID).Error; err != nil {
		return errors.NotFound(err)
	}
	if err := usable(action, now, countsTries); err != nil {
		return err
	}
	return errors.NotFound(gorm.ErrRecordNotFound)
}

func userActionResetPassword(
	db *gorm.DB,
	usersParams *models.UsersParams,
	user entities.User,
	password string,
	now time.Time,
) error {
	passwd := crypt.HashPassword(
		password,
		usersParams.GetArgon2Params(),
		usersParams.GetPasswordSalt(),
	)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", passwd).Error; err != nil {
			return err
		}
		return revokeUserSessions(tx, user.ID, "", entities.SessionRevokedPasswordReset, now)
	})
	if err != nil {
		return errors.DBError(fmt.Errorf("userActionResetPassword: %w", err))
	}
	return nil
}

type UserActionStatusIn struct {
	Realm  string
	Login  string
	Action string
}

type UserActionStatusOut struct {
	Realm       string
	Login       string
	Action      string
	Data        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ValidatedAt *time.Time
	ExpiresAt   *time.Time
	Attempts    int
	MaxAttempts int
}

func UserActionStatus(
	db *gorm.DB,
	in UserActionStatusIn,
) ([]UserActionStatusOut, error) {
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "realm_name", in.Realm)
		return []UserActionStatusOut{}, errors.BadRequest(err)
	}
	if err := KindOf(realm).AssertPasswordFlows(); err != nil {
		return []UserActionStatusOut{}, err
	}
	user := entities.User{}
	if err := db.First(&user, "login = ? AND realm_id = ?", in.Login, realm.ID).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "login", in.Login, "realm_name", in.Realm)
		return []UserActionStatusOut{}, errors.BadRequest(err)
	}
	actions := []entities.UserAction{}
	actionRes := db.
		Where("user_id = ? AND realm_id = ? AND action = ?", user.ID, realm.ID, in.Action).
		Limit(50).
		Order("id DESC").
		Find(&actions)
	if actionRes.Error != nil {
		return []UserActionStatusOut{}, errors.NotFound(actionRes.Error)
	}
	return dt.SliceTransform(actions, func(ua entities.UserAction) UserActionStatusOut {
		return UserActionStatusOut{
			Realm:       realm.Name,
			Login:       user.Login,
			Action:      ua.Action,
			Data:        ua.Data,
			CreatedAt:   ua.CreatedAt,
			UpdatedAt:   ua.UpdatedAt,
			ValidatedAt: ua.ValidatedAt,
			ExpiresAt:   ua.ExpiresAt,
			Attempts:    ua.Attempts,
			MaxAttempts: ua.MaxAttempts,
		}
	}), nil
}
