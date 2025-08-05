package boot

import (
	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/pkg/plugins"
	"github.com/monkeydioude/goauth/pkg/tools/result"
)

// layoutBoot returns handlers and entity related config.
// Those configs will we brought inside http handlers.
func LayoutBoot(
	dbentity []any,
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) result.R[handlers.Layout] {
	dbRes := PostgreSQLBoot(dbentity...)
	if dbRes.IsErr() {
		return result.Error[handlers.Layout](dbRes.Error)
	}
	userParams := usersParamsBoot(loginConstraints, passwordConstraints)
	gorm := dbRes.Result()
	gormSetupHydrate(gorm, userParams)
	return result.Ok(&handlers.Layout{
		DB:         gorm,
		JWTFactory: JwtFactoryBoot(gorm),
		UserParams: userParams,
		Plugins:    &plugins.Plugins,
	})
}
