package services

import (
	stdErr "errors"
	"github.com/monkeydioude/goauth/v2/internal/config/logs"
	"log/slog"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"

	"gorm.io/gorm"
)

func iCanDoUserEditPassword(
	factory *JWTFactory,
	db *gorm.DB,
	editEntity *entities.EditUserPayload,
) bool {
	return factory != nil && db != nil && editEntity != nil && editEntity.NewPassword != nil && editEntity.Password != ""
}

func iCanDoUserEditLogin(
	factory *JWTFactory,
	db *gorm.DB,
	editEntity *entities.EditUserPayload,
) bool {
	return factory != nil && db != nil && editEntity != nil && editEntity.NewLogin != nil && editEntity.Password != ""
}

// assertPasswordFlows refuses the password flows of a user whose realm has none.
func assertPasswordFlows(db *gorm.DB, user *entities.User) error {
	var realm entities.Realm
	if err := db.First(&realm, "id = ?", user.RealmID).Error; err != nil {
		if stdErr.Is(err, gorm.ErrRecordNotFound) {
			// no realm, no flows
			return errors.Forbidden(stdErr.New(consts.ERR_FORBIDDEN_BY_REALM_KIND))
		}
		return errors.DBError(err)
	}
	return KindOf(realm).AssertPasswordFlows()
}

func UserEditPassword(
	tokenWithBearer string,
	factory *JWTFactory,
	db *gorm.DB,
	editEntity *entities.EditUserPayload,
) error {
	if !iCanDoUserEditPassword(factory, db, editEntity) {
		return errors.InternalServerError(stdErr.New(consts.ERR_INTERNAL_ERROR))
	}
	err := editEntity.UserParams.AssertPassword(*editEntity.NewPassword, &editEntity.Password)
	if err != nil {
		return errors.BadRequest(err)
	}
	// only an active session may change the password
	jwt, err := AuthenticateBearer(logs.DBContext(db), tokenWithBearer, *factory)
	if err != nil {
		return err
	}
	signedPasswd := crypt.HashPassword(
		editEntity.Password,
		editEntity.UserParams.GetArgon2Params(),
		editEntity.UserParams.GetPasswordSalt(),
	)
	user := &entities.User{
		ID:       jwt.Claims.UID,
		Password: signedPasswd,
	}
	if err := db.First(user, "id = ? AND password = ?", jwt.Claims.UID, signedPasswd).Error; err != nil {
		return errors.Unauthorized(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
	}
	if user.ID == 0 {
		return errors.BadRequest(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
	}
	if err := assertPasswordFlows(db, user); err != nil {
		return err
	}

	user.Password = *editEntity.NewPassword

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}
		// the session of the access token that authorized the change goes on
		return revokeUserSessions(tx, user.ID, jwt.Claims.SID, entities.SessionRevokedPasswordChanged, factory.TimeFn())
	})
}

func UserEditLogin(
	tokenWithBearer string,
	factory *JWTFactory,
	db *gorm.DB,
	editEntity *entities.EditUserPayload,
) error {
	if !iCanDoUserEditLogin(factory, db, editEntity) {
		return errors.InternalServerError(stdErr.New(consts.ERR_INTERNAL_ERROR))
	}

	// only an active session may change the login
	jwt, err := AuthenticateBearer(logs.DBContext(db), tokenWithBearer, *factory)
	if err != nil {
		return err
	}

	signedPasswd := crypt.HashPassword(
		editEntity.Password,
		editEntity.UserParams.GetArgon2Params(),
		editEntity.UserParams.GetPasswordSalt(),
	)
	user := &entities.User{}
	if err := db.Find(user, "id = ? AND password = ?", jwt.Claims.UID, signedPasswd).Error; err != nil {
		return errors.InternalServerError(err)
	}
	slog.InfoContext(logs.DBContext(db), "trying to change login", "login_before", user.Login, "login_after", *editEntity.NewLogin)
	if user.Login == *editEntity.NewLogin {
		return nil
	}
	if user.ID == 0 {
		return errors.BadRequest(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
	}
	if err := assertPasswordFlows(db, user); err != nil {
		return err
	}
	err = editEntity.UserParams.AssertLogin(*editEntity.NewLogin, &user.Login)
	if err != nil {
		return errors.BadRequest(err)
	}

	// user.Login = *editEntity.NewLogin
	return db.
		Model(user).
		Where("id = ? AND login = ? AND password = ?", jwt.Claims.UID, user.Login, signedPasswd).
		Update("login", *editEntity.NewLogin).Error
	// return db.Save(&user).Error
}
