package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/http/rpc"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRPCCanLogin(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCCanLogin@test.com"
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef }
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second
	trialJWT, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	assert.NoError(t, err)

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	res, err := client.Login(
		ctx,
		&v1.UserRequest{
			Login:    login,
			Password: passwd,
		},
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+trialJWT.Token, cookie.Value)
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, "Ok", res.Message)
}

func TestRPCCantLoginUnmatchingLogin(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCCantLoginUnmatchingLogin@test.com"
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	res, err := client.Login(
		ctx,
		&v1.UserRequest{
			Login:    login,
			Password: "fake-password",
		},
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.Error(t, err)
	assert.Equal(t, "", cookie.Value)
	assert.Equal(t, int32(401), res.Code)
	assert.NotEqual(t, "Ok", res.Message)
}
