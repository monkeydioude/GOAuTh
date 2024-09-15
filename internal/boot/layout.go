package boot

import (
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/constraints"
	"GOAuTh/pkg/tools/result"
)

// layoutBoot returns handlers and entities related config.
// Those configs will we brought inside http handlers.
func layoutBoot(
	dbEntities []any,
	loginConstraint constraints.EntityField,
) result.R[handlers.Layout] {
	dbRes := postgreSQLBoot(dbEntities...)
	if dbRes.IsErr() {
		return result.Error[handlers.Layout](dbRes.Error)
	}
	return result.Ok(&handlers.Layout{
		DB:              dbRes.Result(),
		LoginConstraint: loginConstraint,
		UserParams:      usersParamsBoot(),
		JWTFactory:      jwtBoot(),
	})
}
