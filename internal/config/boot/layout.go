package boot

import (
	"os"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/v2/pkg/http/request"
	"github.com/monkeydioude/goauth/v2/pkg/plugins"
	"github.com/monkeydioude/goauth/v2/pkg/tools/result"
)

// layoutBoot returns handlers and entity related config.
// Those configs will we brought inside http handlers.
func LayoutBoot(
	dbentity []any,
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) result.R[handlers.Layout] {
	trustedProxies, err := request.ParseTrustedProxies(os.Getenv(consts.TRUSTED_PROXIES))
	if err != nil {
		return result.Error[handlers.Layout](err)
	}
	sessionEnv, err := SessionBoot()
	if err != nil {
		return result.Error[handlers.Layout](err)
	}
	dbRes := PostgreSQLBoot(dbentity...)
	if dbRes.IsErr() {
		return result.Error[handlers.Layout](dbRes.Error)
	}
	if err := dropLegacyColumns(dbRes.Result()); err != nil {
		return result.Error[handlers.Layout](err)
	}
	if err := dropLegacyIndexes(dbRes.Result()); err != nil {
		return result.Error[handlers.Layout](err)
	}
	userParams := UsersParamsBoot(loginConstraints, passwordConstraints)
	gorm := dbRes.Result()
	gormSetupHydrate(gorm, userParams)
	atf, rtf := JwtFactoryBoot(gorm, sessionEnv.TTL())
	return result.Ok(&handlers.Layout{
		DB:                  gorm,
		AccessTokenFactory:  atf,
		RefreshTokenFactory: rtf,
		UserParams:          userParams,
		Plugins:             &plugins.Plugins,
		TrustedProxies:      trustedProxies,
		MaxActiveSessions:   sessionEnv.MaxActive,
		SessionReuseGrace:   sessionEnv.ReuseGrace(),
	})
}
