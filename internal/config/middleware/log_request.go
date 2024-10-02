package middleware

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func APILogRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("[%s] incoming API call on %s", r.Header.Get(consts.X_REQUEST_ID_LABEL), r.URL)
}

func GRPCLogRequest(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	xReqId, ok := rpc.GetFirstIncomingMeta(ctx, consts.X_REQUEST_ID_LABEL)
	if !ok {
		xReqId = consts.NO_X_REQUEST_ID
	}
	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("[%s] incoming RPC call on %s, with metadata: %+v\n", xReqId, info.FullMethod, md)
	return handler(ctx, req)
}
