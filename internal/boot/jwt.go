package boot

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/entities"
	"GOAuTh/pkg/crypt"
	"os"
	"time"
)

func jwtBoot() *entities.JWTFactory {
	return &entities.JWTFactory{
		SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		ExpiresIn:     consts.JWTExpiresIn,
		RefreshesIn:   consts.JWTRefreshesIn,
		TimeFn: func() time.Time {
			return time.Now()
		},
	}
}
