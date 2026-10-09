package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	goauthLogs "github.com/monkeydioude/goauth/v2/internal/config/logs"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRedactMetadataHidesTokensAndCookies(t *testing.T) {
	md := metadata.Pairs(
		"set-cookie", "Authorization=Bearer access-secret",
		"set-cookie", "Refresh=refresh-secret",
		"authorization", "Bearer header-secret",
		"cookie", "Refresh=cookie-secret",
		"x-request-id", "req-1",
	)

	trial := redactMetadata(md)

	assert.Equal(t, []string{redactedValue, redactedValue}, trial.Get("set-cookie"))
	assert.Equal(t, []string{redactedValue}, trial.Get("authorization"))
	assert.Equal(t, []string{redactedValue}, trial.Get("cookie"))
	assert.Equal(t, []string{"req-1"}, trial.Get("x-request-id"))
	// the request's own metadata is left untouched
	assert.Equal(t, []string{"Authorization=Bearer access-secret", "Refresh=refresh-secret"}, md.Get("set-cookie"))
}

func TestGRPCLogRequestLogsNoTokens(t *testing.T) {
	var logs bytes.Buffer
	defaultOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(defaultOutput) })

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"set-cookie", "Refresh=refresh-secret",
		"x-request-id", "req-1",
	))
	called := false
	_, err := GRPCLogRequest(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/v1.JWT/Refresh"}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
	assert.Contains(t, logs.String(), "/v1.JWT/Refresh")
	assert.Contains(t, logs.String(), redactedValue)
	assert.NotContains(t, logs.String(), "refresh-secret")
}

// captureJSONLogs makes the default logger write JSON, with the X-Request-ID, into the returned buffer.
func captureJSONLogs(t *testing.T) *bytes.Buffer {
	var logs bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(goauthLogs.NewHandler(slog.NewJSONHandler(&logs, nil))))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	return &logs
}

func decodeLogLines(t *testing.T, logs *bytes.Buffer) []map[string]any {
	var lines []map[string]any
	for _, raw := range bytes.Split(bytes.TrimSpace(logs.Bytes()), []byte("\n")) {
		line := map[string]any{}
		assert.NoError(t, json.Unmarshal(raw, &line))
		lines = append(lines, line)
	}
	return lines
}

// chain runs the interceptors in the order the servers chain them.
func chain(ctx context.Context, method string, handler grpc.UnaryHandler) (any, error) {
	info := &grpc.UnaryServerInfo{FullMethod: method}
	return GRPXRequestID(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return GRPCLogRequest(ctx, req, info, handler)
	})
}

func TestGRPCLogRequestLogsErrorOutcomeWithXRequestID(t *testing.T) {
	logs := captureJSONLogs(t)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "req-1"))
	_, err := chain(ctx, "/v1.JWT/Status", func(ctx context.Context, _ any) (any, error) {
		slog.InfoContext(ctx, "inside the handler")
		return nil, status.Error(codes.Unauthenticated, "token expired")
	})

	assert.Error(t, err)
	lines := decodeLogLines(t, logs)
	assert.Len(t, lines, 3)
	for _, line := range lines {
		assert.Equal(t, "req-1", line["X-Request-ID"])
	}
	assert.Equal(t, "Unauthenticated", lines[2]["code"])
	assert.Equal(t, "token expired", lines[2]["error"])
}

func TestGRPCLogRequestGeneratesOneXRequestIDWhenMissing(t *testing.T) {
	logs := captureJSONLogs(t)

	_, err := chain(metadata.NewIncomingContext(context.Background(), metadata.MD{}), "/v1.JWT/Status", func(context.Context, any) (any, error) {
		return nil, nil
	})

	assert.NoError(t, err)
	lines := decodeLogLines(t, logs)
	assert.Len(t, lines, 2)
	assert.NotEmpty(t, lines[0]["X-Request-ID"])
	assert.Equal(t, lines[0]["X-Request-ID"], lines[1]["X-Request-ID"])
}

func TestAPIXRequestIDReachesTheLogs(t *testing.T) {
	logs := captureJSONLogs(t)

	app := APIXRequestID(APILogRequest(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})))
	req := httptest.NewRequest(http.MethodGet, "/identity/v1/jwt/status", nil)
	req.Header.Set("X-Request-ID", "req-2")
	app.ServeHTTP(httptest.NewRecorder(), req)

	lines := decodeLogLines(t, logs)
	assert.Len(t, lines, 2)
	for _, line := range lines {
		assert.Equal(t, "req-2", line["X-Request-ID"])
	}
	assert.Equal(t, float64(http.StatusUnauthorized), lines[1]["status"])
}
