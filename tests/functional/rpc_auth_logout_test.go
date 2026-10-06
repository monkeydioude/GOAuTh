package functional

import (
	"context"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
)

func TestRPCLogoutRevokesEveryUserSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCLogoutRevokesEveryUserSession@test.com"
	user := newLoginUser(t, gormDB, login)
	tokenA, sessionA := loginSession(t, layout, login)
	tokenB, sessionB := loginSession(t, layout, login)
	otherLogin := "TestRPCLogoutLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	otherToken, otherSession := loginSession(t, layout, otherLogin)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAuthClient(conn)

	// the realm must be the user's
	_, err := client.Logout(context.Background(), &v1.LogoutRequest{Uid: int32(user.ID), Realm: "not-" + login})
	assert.NoError(t, err)
	assert.False(t, findSessionRow(t, gormDB, sessionA.ID.String()).DeletedAt.Valid)

	res, err := client.Logout(context.Background(), &v1.LogoutRequest{Uid: int32(user.ID), Realm: login})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	// every session of the user is revoked, and refuses to refresh
	for _, session := range []entities.Session{sessionA, sessionB} {
		stored := findSessionRow(t, gormDB, session.ID.String())
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), stored.RevokedReason)
	}
	assert.Equal(t, 401, refreshOverHTTP(t, layout, tokenA).Code)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, tokenB).Code)

	// another user's session is left alone
	assert.False(t, findSessionRow(t, gormDB, otherSession.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, otherToken).Code)
}
