package boot

import (
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/models"
	"GOAuTh/pkg/crypt"

	"gorm.io/gorm"
)

func gormSetupHydrate(
	db *gorm.DB,
	userParams *models.UsersParams,
) {
	db.Callback().Create().Before("gorm:create").Register("before_create_password", func(tx *gorm.DB) {
		if user, ok := tx.Statement.Dest.(*entities.User); ok {
			user.Password = crypt.HashPassword(
				user.Password,
				userParams.GetArgon2Params(),
				userParams.GetPasswordSalt(),
			)
		}
	})

	db.Callback().Update().Before("gorm:update").Register("before_update_password", func(tx *gorm.DB) {
		if user, ok := tx.Statement.Dest.(*entities.User); ok {
			user.Password = crypt.HashPassword(
				user.Password,
				userParams.GetArgon2Params(),
				userParams.GetPasswordSalt(),
			)
		}
	})
}
