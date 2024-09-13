package boot

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/entities"
	"os"
)

func usersParamsBoot() *entities.UsersParams {
	return &entities.UsersParams{
		PasswdSalt:   []byte(os.Getenv(consts.PASSWD_SALT)),
		Argon2params: consts.Argon2,
	}
}
