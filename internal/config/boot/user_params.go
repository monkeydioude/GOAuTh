package boot

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/internal/domain/models"
	"os"
)

func usersParamsBoot(
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) *models.UsersParams {
	return &models.UsersParams{
		PasswdSalt:          []byte(os.Getenv(consts.PASSWD_SALT)),
		Argon2params:        consts.Argon2,
		LoginConstraints:    loginConstraints,
		PasswordConstraints: passwordConstraints,
	}
}
