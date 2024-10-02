package middleware

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"

	"google.golang.org/grpc"
)

func APIXRequestID(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xRequestID := consts.NO_X_REQUEST_ID
		tmpXReqID := r.Header.Get(consts.X_REQUEST_ID_LABEL)
		if tmpXReqID != "" {
			xRequestID = tmpXReqID
		} else {
			r.Header.Add(consts.X_REQUEST_ID_LABEL, xRequestID)
		}
		handler.ServeHTTP(w, r)
		w.Header().Add(consts.X_REQUEST_ID_LABEL, xRequestID)
	})
}

func GRPXRequestID(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	xReqID, ok := rpc.GetFirstIncomingMeta(ctx, consts.X_REQUEST_ID_LABEL)
	if !ok {
		xReqID = consts.NO_X_REQUEST_ID
		ctx = rpc.WriteIncomingMetas(ctx, [2]string{consts.X_REQUEST_ID_LABEL, consts.NO_X_REQUEST_ID})
	}

	ctx = rpc.WriteOutgoingMetas(ctx, [2]string{consts.X_REQUEST_ID_LABEL, xReqID})
	return handler(ctx, req)
}
