package services

import (
	go_errors "errors"
	"fmt"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
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
	// a person signs up with a password, whatever the configured constraints
	if user.Password == "" {
		return errors.UnprocessableEntity(go_errors.New(consts.ERR_INVALID_INPUT_PASSWORD))
	}
	if err := userParams.AssertAllConstraints(user.Login, nil, user.Password, nil); err != nil {
		return errors.UnprocessableEntity(err)
	}
	var realm entities.Realm
	if err := db.Where("name = ?", user.RealmName).First(&realm).Error; err != nil {
		slog.ErrorContext(logs.DBContext(db), err.Error(), "realm_name", user.RealmName)
		return errors.BadRequest(err)
	}
	if err := KindOf(realm).AssertPasswordFlows(); err != nil {
		return err
	}
	// a login is unique within its realm only
	tmp_u := &entities.User{}
	res := db.First(tmp_u, "login = ? AND realm_id = ?", user.Login, realm.ID)
	if res.Error == nil && tmp_u.ID != 0 {
		slog.ErrorContext(logs.DBContext(db), consts.ERR_USER_ALREADY_EXIST)
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
	// a realm without password flows never logs in, whatever password is stored
	if user.Realm == nil || KindOf(*user.Realm).AssertPasswordFlows() != nil {
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

// AuthDeactivate closes the account the way its realm's kind says: soft-deleted,
// its sessions revoked. An account already gone is a no-op. actor is who asked,
// as the consumer names them; a service account requires one.
func AuthDeactivate(
	uid uint,
	actor string,
	db *gorm.DB,
	now time.Time,
) error {
	if db == nil {
		return go_errors.New("nil pointer(s) in AuthDeactivate param")
	}
	var user entities.User
	if err := db.First(&user, uid).Error; err != nil {
		if go_errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.DBError(err)
	}
	// a realm gone leaves its accounts as they were made: human
	var realm entities.Realm
	if err := db.First(&realm, "id = ?", user.RealmID).Error; err != nil && !go_errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.DBError(err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return KindOf(realm).Delete(tx, &user, actor, now)
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
