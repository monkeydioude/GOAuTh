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

func NewIncomingMetas(ctx context.Context, kvs ...[2]string) context.Context {
	md := metadata.New(nil)
	for _, kv := range kvs {
		md.Set(kv[0], kv[1])
	}

	return metadata.NewIncomingContext(ctx, md)
}

func GetFirstMetadata(md metadata.MD, key string) (string, bool) {
	if md == nil {
		return "", false
	}
	values := md.Get(key)
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func GetFirstIncomingMeta(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	return GetFirstMetadata(md, key)
}
