package boot

import (
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/constraints"
	"GOAuTh/pkg/tools/result"

	"github.com/joho/godotenv"
)

type Settings struct {
	Layout *handlers.Layout
	Api    *Api
}

// Please will sincerely try to boot every mandatory routines,
// just because you asked politely :)
func Please(
	dbEntities []any,
	loginConstraint constraints.EntityField,
) result.R[Settings] {
	err := godotenv.Load()
	if err != nil {
		return result.Error[Settings](err)
	}
	layout := LayoutBoot(dbEntities, loginConstraint)
	if layout.IsErr() {
		return result.Error[Settings](layout.Error)
	}
	return result.Ok(&Settings{
		Layout: layout.Result(),
		Api:    apiBoot(),
	})
}
