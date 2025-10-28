package main

import (
	"context"
	"log"
	"net"
	"syscall"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/boot"
	"github.com/monkeydioude/goauth/internal/config/middleware"
	"github.com/monkeydioude/goauth/internal/domain/entities/constraints"
	v1 "github.com/monkeydioude/goauth/pkg/grpc/v1"
	"github.com/oklog/run"
	"google.golang.org/grpc"
)

func grpcHandlers(server *grpc.Server, layout *handlers.Layout) {
	v1.RegisterJWTServer(server, v1.NewJWTRPCHandler(layout))
	v1.RegisterAuthServer(server, v1.NewAuthRPCHandler(layout))
	v1.RegisterUserServer(server, v1.NewUserRPCHandler(layout))
}

func setupGRPCServer(settings *boot.Settings) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", settings.Grpc.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.GRPCLogRequest,
		middleware.GRPXRequestID,
	))
	grpcHandlers(server, settings.Layout)
	return server, lis
}

func main() {
	res := bootPlease(
		[]constraints.LoginConstraint{constraints.EmailConstraint},
		[]constraints.PasswordConstraint{constraints.PasswordSafetyConstraint},
	)
	if res.IsErr() {
		log.Fatal(res.Error)
	}

	settings := res.Result()

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

	// Signals handling, for graceful stop
	runGroup.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	if err := runGroup.Run(); err != nil {
		log.Fatal(err)
	}
}
