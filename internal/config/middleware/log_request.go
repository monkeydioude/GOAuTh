package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
		slog.InfoContext(r.Context(), ">>> API call", "method", r.Method, "path", r.URL.Path)
		rec := &responseRecorder{rw: w, status: 200}
		start := time.Now()
		handler.ServeHTTP(rec, r)
		slog.InfoContext(r.Context(), "<<< API call", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start).String())
	})
}

func GRPCLogRequest(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	slog.InfoContext(ctx, ">>> RPC call", "method", info.FullMethod, "metadata", redactMetadata(md))
	start := time.Now()
	res, err := handler(ctx, req)
	elapsed := time.Since(start).String()
	if err != nil {
		st := status.Convert(err)
		slog.WarnContext(ctx, "<<< RPC call", "method", info.FullMethod, "code", st.Code().String(), "error", st.Message(), "duration", elapsed)
		return res, err
	}
	slog.InfoContext(ctx, "<<< RPC call", "method", info.FullMethod, "code", "OK", "duration", elapsed)
	return res, err
}

const redactedValue = "[redacted]"

// sensitiveMetadataKeys carry tokens or cookies, so their values never reach the logs.
var sensitiveMetadataKeys = []string{"authorization", "cookie", rpc.SetCookieLabel}

func redactMetadata(md metadata.MD) metadata.MD {
	redacted := md.Copy()
	for _, key := range sensitiveMetadataKeys {
		for i := range redacted[key] {
			redacted[key][i] = redactedValue
		}
	}
	return redacted
}
