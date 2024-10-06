package middleware

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type responseRecorder struct {
	rw     http.ResponseWriter
	status int
}

func (r *responseRecorder) Header() http.Header {
	return r.rw.Header()
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	return r.rw.Write(data)
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.rw.WriteHeader(code)
}

func APILogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] >>> API call on %s", r.Header.Get(consts.X_REQUEST_ID_LABEL), r.URL)
		rec := &responseRecorder{rw: w, status: 200}
		handler.ServeHTTP(rec, r)
		log.Printf("[%s] <<< %d on API %s", r.Header.Get(consts.X_REQUEST_ID_LABEL), rec.status, r.URL)
	})
}

func GRPCLogRequest(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	xReqId, ok := rpc.GetFirstIncomingMeta(ctx, consts.X_REQUEST_ID_LABEL)
	if !ok {
		xReqId = uuid.NewString()
	}
	md, _ := metadata.FromIncomingContext(ctx)
	log.Printf("[%s] >>> RPC call on %s, with metadata: %+v\n", xReqId, info.FullMethod, md)
	return handler(ctx, req)
}
