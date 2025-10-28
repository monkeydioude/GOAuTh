package boot

import (
	"os"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/internal/domain/models"
)

func UsersParamsBoot(
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
