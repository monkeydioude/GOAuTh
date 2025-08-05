package boot

import (
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/pkg/crypt"

	"gorm.io/gorm"
)

func signPassword(userParams *models.UsersParams) func(*gorm.DB) {
	return func(tx *gorm.DB) {
		if user, ok := tx.Statement.Dest.(*entities.User); ok {
			user.Password = crypt.HashPassword(
				user.Password,
				userParams.GetArgon2Params(),
				userParams.GetPasswordSalt(),
			)
		}
	}
}

func gormSetupHydrate(
	db *gorm.DB,
	userParams *models.UsersParams,
) {
	db.Callback().Create().Before("gorm:create").Register("before_create_password", signPassword(userParams))
	db.Callback().Update().Before("gorm:update").Register("before_update_password", signPassword(userParams))
}
