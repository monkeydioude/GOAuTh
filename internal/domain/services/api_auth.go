package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/internal/domain/models"
	"GOAuTh/pkg/errors"
	go_errors "errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

func AuthSignup(
	user *entities.User,
	constraint constraints.EntityField,
	db *gorm.DB,
) error {
	if err := constraint(user.Login); err != nil {
		return errors.UnprocessableEntity(err)
	}

	if res := db.Save(user); res.Error != nil {
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
		return http.Cookie{}, errors.Unauthorized(go_errors.New("login and password don't match"))
	}
	sign, err := JWTFactory.GenerateToken(user.IntoClaims())
	if err != nil {
		return http.Cookie{}, errors.InternalServerError(err)
	}
	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  sign.GetToken(),
		MaxAge: int(sign.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}
