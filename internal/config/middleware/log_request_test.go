package middleware

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
