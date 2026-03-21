package boot

import (
	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/v2/pkg/tools/result"

	"github.com/joho/godotenv"
)

type Settings struct {
	Layout *handlers.Layout
	Api    *Api
	Grpc   *Grpc
}

// Please will sincerely try to boot every mandatory routines,
// just because you asked politely :)
func Please(
	dbentity []any,
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) result.R[Settings] {
	godotenv.Load()
	layout := LayoutBoot(dbentity, loginConstraints, passwordConstraints)
	if layout.IsErr() {
		return result.Error[Settings](layout.Error)
	}
	return result.Ok(&Settings{
		Layout: layout.Result(),
		Api:    apiBoot(),
		Grpc:   GrpcBoot(),
	})
}
