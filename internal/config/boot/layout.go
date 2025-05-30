package boot

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/pkg/plugins"
	"GOAuTh/pkg/tools/result"
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
