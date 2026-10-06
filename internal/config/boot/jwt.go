package boot

import (
	"os"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"

	"gorm.io/gorm"
)

// JwtFactoryBoot returns the access and refresh token factories.
// A refresh token lives as long as its session: sessionTTL.
func JwtFactoryBoot(db *gorm.DB, sessionTTL time.Duration) (*services.JWTFactory, *services.JWTFactory) {
	return &services.JWTFactory{
			SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
			ExpiresIn:     consts.AccessTokenExpiresIn,
			Type:          consts.AuthorizationCookie,
			TimeFn: func() time.Time {
				return time.Now()
			},
			RevocationCheckerFn: func(uid uint, timeFn func() time.Time) (bool, error) {
				return services.IsLoginRevoked[crypt.JWTDefaultClaims, entities.User](uid, db, timeFn())
			},
		}, &services.JWTFactory{
			SigningMethod: crypt.HS256(os.Getenv(consts.JWT_SECRET)),
			ExpiresIn:     sessionTTL,
			Type:          consts.RefreshTokenCookie,
			TimeFn: func() time.Time {
				return time.Now()
			},
			RevocationCheckerFn: func(uid uint, timeFn func() time.Time) (bool, error) {
				return services.IsLoginRevoked[crypt.JWTDefaultClaims, entities.User](uid, db, timeFn())
			},
		}
}
