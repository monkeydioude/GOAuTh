package functional

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/pkg/crypt"
	v1 "github.com/monkeydioude/goauth/pkg/grpc/v1"
	"github.com/monkeydioude/goauth/pkg/http/rpc"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRPCCanGetAValidTokensStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	login := "TestRPCCanGetAValidTokensStatus@test.com"
	jwt, err := layout.AccessTokenFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID:   123,
		Realm: login,
	})
	assert.NoError(t, err)
	client := v1.NewJWTClient(conn)
	ctx := metadata.NewOutgoingContext(context.Background(), rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	{
		_, err := client.Status(
			ctx,
			&v1.StatusIn{},
		)
		assert.NoError(t, err)
	}
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
	{
		res, err := client.Status(
			ctx,
			&v1.StatusIn{},
		)
		assert.Nil(t, res)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, "invalid token format", st.Message())
		assert.Equal(t, http.StatusBadRequest, int(st.Code()))
	}
}

func TestRPCWontValidateAnExpiredToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewJWTClient(conn)
	// ensuring factory's timeline
	layout.AccessTokenFactory.ExpiresIn = 3 * time.Second
	jwt, err := layout.AccessTokenFactory.GenerateToken(crypt.JWTDefaultClaims{
		Realm: "TestRPCWontValidateAnExpiredToken@test.com",
		UID:   123,
	})
	assert.NoError(t, err)
	timeRef := layout.AccessTokenFactory.TimeFn()
	// sending the factory into the futur, making sure we are past refresh time
	layout.AccessTokenFactory.TimeFn = func() time.Time {
		return timeRef.Add(12 * time.Second)
	}
	ctx := metadata.NewOutgoingContext(context.Background(), rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	{
		res, err := client.Status(
			ctx,
			&v1.StatusIn{},
		)
		assert.Nil(t, res)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, "TokenExpiredError", st.Message())
		assert.Equal(t, http.StatusUnauthorized, int(st.Code()))
	}
}
