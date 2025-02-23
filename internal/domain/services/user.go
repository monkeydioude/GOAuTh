package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/errors"
	stdErr "errors"
	"log/slog"

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
	jwtRes := GetJWTFromBearer(tokenWithBearer, factory)
	if jwtRes.IsErr() {
		return jwtRes.Error
	}
	jwt := jwtRes.Result()
	if !JWTClaimsValidation(jwt.Claims) {
		return errors.Unauthorized(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
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

	user.Password = *editEntity.NewPassword

	return db.Save(user).Error
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

	jwtRes := GetJWTFromBearer(tokenWithBearer, factory)
	if jwtRes.IsErr() {
		return jwtRes.Error
	}
	jwt := jwtRes.Result()
	if !JWTClaimsValidation(jwt.Claims) {
		return errors.Unauthorized(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
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
	slog.Info("trying to change login", "login_before", user.Login, "login_after", *editEntity.NewLogin)
	if user.Login == *editEntity.NewLogin {
		return nil
	}
	if user.ID == 0 {
		return errors.BadRequest(stdErr.New(consts.ERR_INVALID_CREDENTIALS))
	}
	err := editEntity.UserParams.AssertLogin(*editEntity.NewLogin, &user.Login)
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
