package services

import (
	"fmt"
	"log/slog"
	"time"

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
}

type UserActionCreateOut struct {
	Data string
}

func UserActionCreate(
	db *gorm.DB,
	in UserActionCreateIn,
	uuidGen func() string,
) (UserActionCreateOut, error) {
	actionType, err := entities.UserActionTypeFromString(in.Action)
	if err != nil {
		return UserActionCreateOut{}, fmt.Errorf("UserActionCreate: %w", err)
	}
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
			return UserActionCreateOut{}, errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
		}
	}
	data := uuidGen()
	dom := entities.UserAction{
		UserID:  user.ID,
		RealmID: realm.ID,
		Action:  actionType,
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
	Login   string
	Realm   string
	Data    string
	Against string
}

func UserActionValidate(
	db *gorm.DB,
	usersParams *models.UsersParams,
	in UserActionValidateIn,
) error {
	var realm entities.Realm
	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", in.Realm)
		return errors.BadRequest(err)
	}
	user := entities.User{}
	if err := db.First(&user, "login = ?", in.Login).Error; err != nil {
		slog.Error(err.Error(), "login", in.Login)
		return errors.BadRequest(err)
	}
	action := entities.UserAction{}
	actionRes := db.First(&action, "user_id = ? AND realm_id = ? AND data = ? AND validated_at IS NULL", user.ID, realm.ID, in.Data)
	if actionRes.Error != nil {
		return errors.NotFound(actionRes.Error)
	}
	var err error
	switch action.Action {
	case entities.UserActionTypePassword:
		err = userActionResetPassword(db, usersParams, user, in.Against)
	case entities.UserActionTypeActivate:
		// if activate account, just do nothing besides updating the validated_at field
		// which comes later in the code
		err = nil
	default:
		err = fmt.Errorf("UserActionValidate: empty or invalid acton")
	}
	if err != nil {
		return errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
	}
	if err := db.Model(&action).Update("validated_at", time.Now()).Error; err != nil {
		return errors.BadRequest(fmt.Errorf("UserActionValidate: %w", err))
	}
	return nil
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

// type UserActionStatusIn struct {
// 	Login  string
// 	Realm  string
// 	Action string
// }

// type UserActionStatusOut struct {
// 	ID          int32
// 	Action      string
// 	CreatedAt   time.Time
// 	UpdatedAt   time.Time
// 	ValidatedAt *time.Time
// }

// func UserActionStatuses(
// 	db *gorm.DB,
// 	in UserActionStatusIn,
// ) ([]UserActionStatusOut, error) {
// 	realm := entities.Realm{}
// 	if err := db.Where("name = ?", in.Realm).First(&realm).Error; err != nil {
// 		slog.Error(err.Error(), "realm_name", in.Realm)
// 		return []UserActionStatusOut{}, errors.BadRequest(err)
// 	}
// 	user := entities.User{}
// 	if err := db.First(&user, "login = ?", in.Login).Error; err != nil {
// 		slog.Error(err.Error(), "login", in.Login)
// 		return []UserActionStatusOut{}, errors.BadRequest(err)
// 	}
// 	action, err := entities.UserActionTypeFromString(in.Action)
// 	if err != nil {
// 		return []UserActionStatusOut{}, fmt.Errorf("UserActionStatus: %w", err)
// 	}
// 	doms := []entities.UserAction{}
// 	actionRes := db.Order(`"user_actions"."id" DESC`).Find(&doms, "user_id = ? AND realm_id = ? AND action = ?", user.ID, realm.ID, action)
// 	if actionRes.Error != nil {
// 		return []UserActionStatusOut{}, errors.NotFound(actionRes.Error)
// 	}
// 	return dt.SliceTransform(doms, func(dom entities.UserAction) UserActionStatusOut {
// 		return UserActionStatusOut{
// 			ID:          int32(dom.ID),
// 			Action:      dom.Action.String(),
// 			CreatedAt:   dom.CreatedAt,
// 			UpdatedAt:   dom.UpdatedAt,
// 			ValidatedAt: dom.ValidatedAt,
// 		}
// 	}), nil
// }
