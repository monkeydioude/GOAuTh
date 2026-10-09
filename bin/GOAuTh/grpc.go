package main

import (
	"log"
	"net"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/boot"
	"github.com/monkeydioude/goauth/v2/internal/config/middleware"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"google.golang.org/grpc"
)

func grpcHandlers(server *grpc.Server, layout *handlers.Layout) {
	v1.RegisterJWTServer(server, v1.NewJWTRPCHandler(layout))
	v1.RegisterAuthServer(server, v1.NewAuthRPCHandler(layout))
	v1.RegisterUserServer(server, v1.NewUserRPCHandler(layout))
	v1.RegisterUserActionServer(server, v1.NewUserActionRPCHandler(layout))
	v1.RegisterSessionServer(server, v1.NewSessionRPCHandler(layout))
	v1.RegisterAccountServer(server, v1.NewAccountRPCHandler(layout))
	v1.RegisterAccessKeyServer(server, v1.NewAccessKeyRPCHandler(layout))
}

func setupGRPCServer(settings *boot.Settings) (*grpc.Server, net.Listener) {
	lis, err := net.Listen("tcp", settings.Grpc.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		middleware.GRPXRequestID,
		middleware.GRPCLogRequest,
	))
	grpcHandlers(server, settings.Layout)
	return server, lis
}
