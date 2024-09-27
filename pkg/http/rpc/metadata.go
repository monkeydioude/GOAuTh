package rpc

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func WriteOutgoingMetas(ctx context.Context, kvs ...[2]string) context.Context {
	for _, kv := range kvs {
		ctx = metadata.AppendToOutgoingContext(ctx, kv[0], kv[1])
	}
	return ctx
}
