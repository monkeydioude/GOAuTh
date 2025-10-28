package main

import (
	"github.com/joho/godotenv"
	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/boot"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	"github.com/monkeydioude/goauth/pkg/tools/result"
)

func layoutBoot(
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) result.R[handlers.Layout] {
	userParams := boot.UsersParamsBoot(loginConstraints, passwordConstraints)
	return result.Ok(&handlers.Layout{
		UserParams: userParams,
		// Plugins:    &plugins.Plugins,
	})
}

func bootPlease(
	loginConstraints []constraints.LoginConstraint,
	passwordConstraints []constraints.PasswordConstraint,
) result.R[boot.Settings] {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	layout := layoutBoot(loginConstraints, passwordConstraints)
	if layout.IsErr() {
		return result.Error[boot.Settings](layout.Error)
	}
	return result.Ok(&boot.Settings{
		Layout: layout.Result(),
		Grpc:   boot.GrpcBoot(),
	})
}
