package boot

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/models"
	"os"
)

func usersParamsBoot() *models.UsersParams {
	return &models.UsersParams{
		PasswdSalt:   []byte(os.Getenv(consts.PASSWD_SALT)),
		Argon2params: consts.Argon2,
	}
}
