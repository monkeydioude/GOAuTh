package boot

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/crypt"
	"os"
	"time"

	"gorm.io/gorm"
)

func jwtFactoryBoot(db *gorm.DB) *services.JWTFactory {
	return &services.JWTFactory{
		SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		ExpiresIn:     consts.JWTExpiresIn,
		RefreshesIn:   consts.JWTRefreshesIn,
		TimeFn: func() time.Time {
			return time.Now()
		},
		RevocationCheckerFn: func(login string, timeFn func() time.Time) (bool, error) {
			return services.IsLoginRevoked[crypt.JWTDefaultClaims, entities.User](login, db, timeFn())
		},
	}
}
