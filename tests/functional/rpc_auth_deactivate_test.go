package functional

import (
	"context"
	"net/http"
	"testing"

	v1 "github.com/monkeydioude/goauth/internal/api/rpc/v1"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/pkg/crypt"
	"github.com/monkeydioude/goauth/pkg/http/rpc"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestRPCCantDeactivateAnUserIfMissingID(t *testing.T) {
	layout, gormDB, _ := setup()
	login := "TestRPCCanDeactivateAnUserByID@test.com"
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
		ID:        1,
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	conn := setupRPC(t, layout)
	t.Cleanup(func() {
		cleanup(layout)
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
		conn.Close()
	})
	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)

	client := v1.NewUserClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	// enforcing JWTFactory time creation date forward in time
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		Realm: realm.Name,
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
	assert.Equal(t, int32(401), res.Code)
	assert.NotEqual(t, "Ok", res.Message)
}

func TestRPCCanDeactivateAnUserByID(t *testing.T) {
	layout, gormDB, _ := setup()
	login := "TestRPCCanDeactivateAnUserByID@test.com"
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
		ID:        1,
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	conn := setupRPC(t, layout)
	t.Cleanup(func() {
		cleanup(layout)
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
		conn.Close()
	})
	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)

	client := v1.NewUserClient(conn)
	ctx := context.Background()

	var headerMD metadata.MD
	// enforcing JWTFactory time creation date forward in time
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID:   1,
		Realm: realm.Name,
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
