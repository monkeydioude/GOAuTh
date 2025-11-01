package main

import (
	"context"
	"log"
	"syscall"

	"github.com/monkeydioude/goauth/internal/config/boot"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"

	// _ "github.com/monkeydioude/goauth/plugins"

	"github.com/oklog/run"
)

func main() {
	res := boot.Please(
		[]any{entities.NewEmptyUser(), &entities.Realm{}},
		[]constraints.LoginConstraint{constraints.EmailConstraint},
		[]constraints.PasswordConstraint{constraints.PasswordSafetyConstraint},
	)
	if res.IsErr() {
		log.Fatal(res.Error)
	}

	settings := res.Result()

	// server definition
	apiServer := setupAPIServer(settings)
	grpcServer, lis := setupGRPCServer(settings)

	// synchronisation of async servers
	runGroup := &run.Group{}

	// RPC goroutine
	runGroup.Add(func() error {
		log.Println("RPC starting on", lis.Addr())
		return grpcServer.Serve(lis)
	}, func(_ error) {
		log.Println("stopping RPC server")
		grpcServer.GracefulStop()
		grpcServer.Stop()
	})

	// JSON API goroutine
	runGroup.Add(func() error {
		// Start the server on port
		log.Println("API starting on", apiServer.Addr)
		return apiServer.ListenAndServe()
	}, func(_ error) {
		log.Println("closing API server")
		if err := apiServer.Close(); err != nil {
			log.Println("failed to stop web server", "err", err)
		}
	})

	// Signals handling, for graceful stop
	runGroup.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	if err := runGroup.Run(); err != nil {
		log.Fatal(err)
	}
}
