package services

import (
	go_errors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/models"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"github.com/google/uuid"
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
	var realm entities.Realm
	if err := db.Where("name = ?", user.RealmName).First(&realm).Error; err != nil {
		slog.Error(err.Error(), "realm_name", user.RealmName)
		return errors.BadRequest(err)
	}
	// a login is unique within its realm only
	tmp_u := &entities.User{}
	res := db.First(tmp_u, "login = ? AND realm_id = ?", user.Login, realm.ID)
	if res.Error == nil && tmp_u.ID != 0 {
		slog.Error(consts.ERR_USER_ALREADY_EXIST)
		return errors.BadRequest(go_errors.New(consts.ERR_USER_ALREADY_EXIST))
	}
	user.RealmID = realm.ID

	if res := db.Omit("realm").Create(user); res.Error != nil {
		return errors.DBError(res.Error)
	}
	return nil
}

// AuthLogin checks the user's credentials and creates a session for this login:
// both tokens carry its id as their sid, and it stores the refresh token's hash.
func AuthLogin(
	user *entities.User,
	client ClientInfo,
	db *gorm.DB,
	usersParams *models.UsersParams,
	accessTokenFactory *JWTFactory,
	refreshTokenFactory *JWTFactory,
	maxActiveSessions int,
) (http.Cookie, http.Cookie, error) {
	if user == nil || db == nil || usersParams == nil || accessTokenFactory == nil || refreshTokenFactory == nil {
		return http.Cookie{}, http.Cookie{}, go_errors.New("nil pointer(s) in AuthLogin param")
	}
	if user.IsRevoked(time.Now()) {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(go_errors.New("user's access was revoked"))
	}
	if err := user.AssertAuth(db, usersParams); err != nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(go_errors.New("InvalidCredentials"))
	}
	sid := uuid.New()
	claims := user.IntoClaims()
	claims.SID = sid.String()
	accessToken, err := accessTokenFactory.GenerateToken(claims)
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.InternalServerError(err)
	}
	refreshToken, err := refreshTokenFactory.GenerateToken(claims)
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.InternalServerError(err)
	}
	now := refreshTokenFactory.TimeFn()
	err = db.Transaction(func(tx *gorm.DB) error {
		return createSession(tx, &entities.Session{
			ID:             sid,
			UserID:         user.ID,
			TokenHash:      crypt.HashToken(refreshToken.GetToken()),
			UserAgent:      client.UserAgent,
			LoginIP:        client.IP,
			LastIP:         client.IP,
			LastConnection: now,
			ExpiresAt:      time.Unix(refreshToken.Claims.Expire, 0),
		}, maxActiveSessions, now)
	})
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.DBError(err)
	}
	return http.Cookie{
			Name:    consts.AuthorizationCookie,
			Value:   "Bearer " + accessToken.GetToken(),
			MaxAge:  int(accessTokenFactory.ExpiresIn.Seconds()),
			Path:    "/",
			Expires: time.Now().Add(accessTokenFactory.ExpiresIn),
		}, http.Cookie{
			Name:    consts.RefreshTokenCookie,
			Value:   refreshToken.GetToken(),
			MaxAge:  int(refreshTokenFactory.ExpiresIn.Seconds()),
			Path:    "/",
			Expires: time.Now().Add(refreshTokenFactory.ExpiresIn),
		}, nil
}

// AuthDeactivate soft-deletes the user and revokes all their sessions.
func AuthDeactivate(
	uid uint,
	db *gorm.DB,
	now time.Time,
) error {
	if db == nil {
		return go_errors.New("nil pointer(s) in AuthDeactivate param")
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&entities.User{}, uid).Error; err != nil {
			return err
		}
		return revokeUserSessions(tx, uid, "", entities.SessionRevokedAccountDeactivated, now)
	})
}

// AuthLogout ends every session of the user of realm, as logout did when a
// user had a single session.
func AuthLogout(
	uid uint,
	realm string,
	db *gorm.DB,
	now time.Time,
) error {
	if db == nil {
		return go_errors.New("nil pointer(s) in AuthLogout param")
	}
	user := db.Model(&entities.User{}).
		Select("id").
		Where("id = ? AND realm_id = (?)", uid, db.Table("realms").Select("id").Where("name = ?", realm))
	return RevokeSessions(db, entities.SessionRevokedLogout, now, "user_id IN (?)", user)
}
