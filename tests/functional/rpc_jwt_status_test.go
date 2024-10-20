package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestRPCCanGetAValidTokensStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	// login := "TestRPCCanGetAValidTokensStatus@test.com"
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
	})
	assert.NoError(t, err)
	client := v1.NewJWTClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, _ := client.Status(
		ctx,
		&v1.Empty{},
	)
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, "Ok", res.Message)
}

func TestRPCWontValidateABadTokenStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewJWTClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer fake-token",
	}))
	res, _ := client.Status(
		ctx,
		&v1.Empty{},
	)
	assert.NotNil(t, res)
	assert.Equal(t, int32(400), res.Code)
	assert.NotEqual(t, "Ok", res.Message)
}

func TestRPCWontValidateAnExpiredToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewJWTClient(conn)
	// ensuring factory's timeline
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: "TestRPCWontValidateAnExpiredToken@test.com",
	})
	assert.NoError(t, err)
	timeRef := layout.JWTFactory.TimeFn()
	// sending the factory into the futur, making sure we are past refresh time
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef.Add(12 * time.Second)
	}
	ctx := metadata.NewOutgoingContext(context.Background(), rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, _ := client.Status(
		ctx,
		&v1.Empty{},
	)
	assert.NotNil(t, res)
	assert.Equal(t, int32(401), res.Code)
	assert.Equal(t, consts.ERR_TOKEN_EXPIRED, res.Message)
}
