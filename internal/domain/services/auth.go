package services

import (
	go_errors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/pkg/errors"

	"gorm.io/gorm"
)

func AuthSignup(
	user *entities.User,
	userParams *models.UsersParams,
	db *gorm.DB,
) error {
	if db == nil {
		return errors.InternalServerError(fmt.Errorf("nil db object"))
	}
	if err := userParams.AssertAllConstraints(user.Login, nil, user.Password, nil); err != nil {
		return errors.UnprocessableEntity(err)
	}
	tmp_u := &entities.User{}
	res := db.First(tmp_u, "login = ?", user.Login)

	if res.Error == nil && tmp_u.ID != 0 {
		slog.Error(consts.ERR_USER_ALREADY_EXIST)
		return errors.BadRequest(go_errors.New(consts.ERR_USER_ALREADY_EXIST))
	}

	var realm entities.Realm
	if err := db.Where("name = ?", user.RealmName).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", user.RealmName)
		return errors.BadRequest(err)
	}
	user.RealmID = realm.ID

	if res := db.Omit("realm").Create(user); res.Error != nil {
		return errors.DBError(res.Error)
	}
	return nil
}

func AuthLogin(
	user *entities.User,
	db *gorm.DB,
	usersParams *models.UsersParams,
	JWTFactory *JWTFactory,
) (http.Cookie, error) {
	if user == nil || db == nil || usersParams == nil || JWTFactory == nil {
		return http.Cookie{}, go_errors.New("nil pointer(s) in AuthLogin param")
	}
	if user.IsRevoked(time.Now()) {
		return http.Cookie{}, errors.Unauthorized(go_errors.New("user's access was revoked"))
	}
	if err := user.AssertAuth(db, usersParams); err != nil {
		return http.Cookie{}, errors.Unauthorized(go_errors.New("InvalidCredentials"))
	}
	sign, err := JWTFactory.GenerateToken(user.IntoClaims())
	if err != nil {
		return http.Cookie{}, errors.InternalServerError(err)
	}
	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  "Bearer " + sign.GetToken(),
		MaxAge: int(sign.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

func AuthDeactivate(
	uid uint,
	db *gorm.DB,
) error {
	if db == nil {
		return go_errors.New("nil pointer(s) in AuthDeactivate param")
	}

	return db.Delete(&entities.User{}, uid).Error
}
