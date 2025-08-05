package boot

import (
	"os"
	"time"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/crypt"

	"gorm.io/gorm"
)

func JwtFactoryBoot(db *gorm.DB) *services.JWTFactory {
	return &services.JWTFactory{
		SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		ExpiresIn:     consts.JWTExpiresIn,
		RefreshesIn:   consts.JWTRefreshesIn,
		TimeFn: func() time.Time {
			return time.Now()
		},
		RevocationCheckerFn: func(uid uint, timeFn func() time.Time) (bool, error) {
			return services.IsLoginRevoked[crypt.JWTDefaultClaims, entities.User](uid, db, timeFn())
		},
	}
}
