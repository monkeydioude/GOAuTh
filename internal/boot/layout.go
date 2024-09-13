package boot

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/constraints"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/tools/result"
	"os"
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
		SigningMethod:   crypt.HS256(os.Getenv(consts.JWT_SECRET)),
		UserParams:      usersParamsBoot(),
	})
}
