package boot

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/entities"
	"GOAuTh/internal/services"
	"GOAuTh/pkg/crypt"
	"os"
	"time"

	"gorm.io/gorm"
)

func jwtBoot(db *gorm.DB) *entities.JWTFactory {
	return &entities.JWTFactory{
		SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		ExpiresIn:     consts.JWTExpiresIn,
		RefreshesIn:   consts.JWTRefreshesIn,
		TimeFn: func() time.Time {
			return time.Now()
		},
		RevocationCheckerFn: func(login string) (bool, error) {
			return services.IsLoginRevoked[crypt.JWTDefaultClaims, entities.DefaultUser](login, db, time.Now())
		},
	}
}
