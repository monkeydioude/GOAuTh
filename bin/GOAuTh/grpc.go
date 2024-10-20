package main

import (
	"GOAuTh/internal/api/handlers"
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/boot"
	"GOAuTh/internal/config/middleware"
	"log"
	"net"

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
