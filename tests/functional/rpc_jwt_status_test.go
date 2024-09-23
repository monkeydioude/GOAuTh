package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/pkg/crypt"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPCCanGetAValidTokensStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	login := "TestRPCCanGetAValidTokensStatus@test.com"
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	assert.NoError(t, err)
	client := v1.NewJWTClient(conn)
	ctx := context.Background()
	res, err := client.Status(ctx, &v1.JWTRequest{
		Token: jwt.Token,
	})
	assert.NoError(t, err)
	assert.Equal(t, res.Code, int32(200))
	assert.Equal(t, res.Message, "Ok")
}

func TestRPCWontValidateABadTokenStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewJWTClient(conn)
	ctx := context.Background()
	res, _ := client.Status(ctx, &v1.JWTRequest{
		Token: "fake-token",
	})
	assert.NotNil(t, res)
	assert.Equal(t, int32(400), res.Code)
	assert.NotEqual(t, "Ok", res.Message)
}
