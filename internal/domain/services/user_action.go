package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/pkg/crypt"
	"github.com/monkeydioude/goauth/pkg/errors"
	"gorm.io/gorm"
)

type UserActionCreateIn struct {
	Login  string
	Realm  string
	Action string
	Data   *string
}

type UserActionCreateOut struct {
	Data string
}

func UserActionCreate(
	db *gorm.DB,
	in UserActionCreateIn,
	uuidGen func() string,
) (UserActionCreateOut, error) {
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", in.Realm)
		return UserActionCreateOut{}, errors.BadRequest(err)
	}
	user := entities.User{}
	if err := db.First(&user, "login = ?", in.Login).Error; err != nil {
		slog.Error(err.Error(), "login", in.Login)
		return UserActionCreateOut{}, errors.BadRequest(err)
	}
	action := entities.UserAction{}
	actionRes := db.First(&action, "user_id = ? AND realm_id = ? AND action = ? AND validated_at IS NULL", user.ID, realm.ID, in.Action)
	if actionRes.Error == nil && action.ID != 0 {
		if err := db.Model(&action).Update("validated_at", time.Now()).Error; err != nil {
			return UserActionCreateOut{}, errors.BadRequest(fmt.Errorf("UserActionCreate: %w", err))
		}
	}
	data := dt.Deref(in.Data)
	if in.Action == entities.UserActionTypePassword || data == "" {
		data = uuidGen()
	}
	dom := entities.UserAction{
		UserID:  user.ID,
		RealmID: realm.ID,
		Action:  in.Action,
		Data:    data,
	}
	if res := db.Create(&dom); res.Error != nil {
		return UserActionCreateOut{}, errors.DBError(res.Error)
	}
	return UserActionCreateOut{
		Data: data,
	}, nil
}

type UserActionValidateIn struct {
	Realm   string
	Data    string
	Against string
}

func UserActionValidate(
	db *gorm.DB,
	usersParams *models.UsersParams,
	in UserActionValidateIn,
) (string, error) {
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", in.Realm)
		return "", errors.BadRequest(err)
	}
	action := entities.UserAction{}
	actionRes := db.First(&action, "realm_id = ? AND data = ? AND validated_at IS NULL", realm.ID, in.Data)
	if actionRes.Error != nil {
		return "", errors.NotFound(actionRes.Error)
	}
	user := entities.User{}
	if err := db.First(&user, "id = ?", action.UserID).Error; err != nil {
		slog.Error(err.Error(), "user_id", action.UserID)
		return "", errors.BadRequest(err)
	}
	var err error
	switch action.Action {
	case entities.UserActionTypePassword:
		err = userActionResetPassword(db, usersParams, user, in.Against)
	}
	if err != nil {
		return "", errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
	}
	if err := db.Model(&action).Update("validated_at", time.Now()).Error; err != nil {
		return "", errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
	}
	return user.Login, nil
}

func userActionResetPassword(
	db *gorm.DB,
	usersParams *models.UsersParams,
	user entities.User,
	password string,
) error {
	passwd := crypt.HashPassword(
		password,
		usersParams.GetArgon2Params(),
		usersParams.GetPasswordSalt(),
	)
	res := db.Model(&user).Update("password", passwd)
	if res.Error != nil {
		return errors.DBError(fmt.Errorf("userActionResetPassword: %w", res.Error))
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
}

func UserActionStatus(
	db *gorm.DB,
	in UserActionStatusIn,
) ([]UserActionStatusOut, error) {
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", in.Realm)
		return []UserActionStatusOut{}, errors.BadRequest(err)
	}
	user := entities.User{}
	if err := db.First(&user, "login = ?", in.Login).Error; err != nil {
		slog.Error(err.Error(), "login", in.Login)
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
		}
	}), nil
}
