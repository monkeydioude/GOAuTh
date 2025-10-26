package main

import (
	"log"
	"net"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/boot"
	"github.com/monkeydioude/goauth/internal/config/middleware"
	v1 "github.com/monkeydioude/goauth/pkg/grpc/v1"

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
