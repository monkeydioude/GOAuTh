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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRPCCanRefreshAValidTokens(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestICanRefreshAValidToken@test.com"
	passwd := "test"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		ID:        11,
		RealmID:   realm.ID,
		RealmName: login,
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	// enforcing date for the jwt generation
	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef }
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		UID:   11,
		Realm: login,
	})
	assert.NoError(t, err)

	// enforcing JWTFactory time creation date forward in time
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef.Add(5 * time.Second) }
	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewJWTClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	ctx = metadata.NewOutgoingContext(ctx, rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, err := client.Refresh(
		ctx,
		&v1.Empty{},
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.NoError(t, err)
	assert.NotEqual(t, "Bearer "+jwt.Token, cookie.Value)
	jwt2, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		UID:   11,
		Realm: login,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+jwt2.Token, cookie.Value)
	assert.NoError(t, err)
	assert.Equal(t, res.Code, int32(200))
	assert.Equal(t, res.Message, "Ok")
}

func TestRPCCanNotRefreshExpiredToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	login := "TestRPCCanNotRefreshExpiredToken@test.com"

	// enforcing date for the jwt generation
	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef }
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID:   12,
		Realm: login,
	})
	// enforcing JWTFactory time creation date forward in time
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef.Add(12 * time.Second) }
	assert.NoError(t, err)
	client := v1.NewJWTClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	ctx = metadata.NewOutgoingContext(ctx, rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, err := client.Refresh(
		ctx,
		&v1.Empty{},
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.Error(t, err)
	assert.Equal(t, "", cookie.Value)
	assert.Equal(t, int32(401), res.Code)
	assert.NotEqual(t, res.Message, "Ok")
}

func TestRPCReturnsSameTokenIfValid(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	conn := setupRPC(t, layout)
	defer conn.Close()
	login := "TestRPCReturnsSameTokenIfValid@test.com"

	// enforcing date for the jwt generation
	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef }
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID:   113,
		Realm: login,
	})
	// enforcing JWTFactory time creation date forward in time
	layout.JWTFactory.TimeFn = func() time.Time { return timeRef.Add(2 * time.Second) }
	assert.NoError(t, err)
	client := v1.NewJWTClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	ctx = metadata.NewOutgoingContext(ctx, rpc.SetCookie(http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "Bearer " + jwt.Token,
	}))
	res, err := client.Refresh(
		ctx,
		&v1.Empty{},
		grpc.Header(&headerMD),
	)
	assert.NoError(t, err)
	cookie, err := rpc.FetchCookie(headerMD, consts.AuthorizationCookie)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+jwt.Token, cookie.Value)
	assert.Equal(t, int32(200), res.Code)
	assert.Equal(t, res.Message, "Ok")
}
