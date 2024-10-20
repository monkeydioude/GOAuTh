package boot

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/pkg/tools/result"

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
	err := godotenv.Load()
	if err != nil {
		return result.Error[Settings](err)
	}
	layout := LayoutBoot(dbentity, loginConstraints, passwordConstraints)
	if layout.IsErr() {
		return result.Error[Settings](layout.Error)
	}
	return result.Ok(&Settings{
		Layout: layout.Result(),
		Api:    apiBoot(),
		Grpc:   grpcBoot(),
	})
}
