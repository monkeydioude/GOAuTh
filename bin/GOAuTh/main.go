package main

import (
	"context"
	"log"
	"syscall"

	"GOAuTh/internal/config/boot"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/pkg/plugins"
	_ "GOAuTh/plugins"

	"github.com/oklog/run"
)

func getSettings() *boot.Settings {
	plgins := &plugins.Plugins
	res := boot.Please(
		[]any{entities.NewEmptyUser()},
		[]constraints.LoginConstraint{constraints.EmailConstraint},
		[]constraints.PasswordConstraint{constraints.PasswordSafetyConstraint},
		plgins,
	)
	if res.IsErr() {
		log.Fatal(res.Error)
	}

	return res.Result()
}

func main() {
	settings := getSettings()
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
