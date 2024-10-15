package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRPCCantDeactivateAnUserIfMissingID(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCCanDeactivateAnUserByID@test.com"
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		ID:        1,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	// enforcing JWTFactory time creation date forward in time
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	assert.NoError(t, err)
	ctx = metadata.NewOutgoingContext(ctx, rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, err := client.Deactivate(
		ctx,
		nil,
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	assert.NoError(t, gormDB.First(&user, 1).Error)
	assert.Equal(t, int32(400), res.Code)
	assert.NotEqual(t, "Ok", res.Message)
}

func TestRPCCanDeactivateAnUserByID(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCCanDeactivateAnUserByID@test.com"
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		ID:        1,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	// enforcing JWTFactory time creation date forward in time
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
		UID:  1,
	})
	assert.NoError(t, err)
	ctx = metadata.NewOutgoingContext(ctx, rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, err := client.Deactivate(
		ctx,
		nil,
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	assert.Error(t, gormDB.First(&user, 1).Error)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.Error(t, err)
	assert.Equal(t, "", cookie.Value)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, "Ok", res.Message)
}
