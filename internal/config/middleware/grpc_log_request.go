package middleware

import (
	"context"
	"log"

	"google.golang.org/grpc"
)

func GRPCLogRequest(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	log.Printf("incoming RPC call on %s\n", info.FullMethod)
	resp, err := handler(ctx, req)
	return resp, err
}
