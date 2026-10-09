package functional

import (
	"context"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
)

func TestRPCLogoutWithoutATokenIsRefused(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCLogoutWithoutATokenIsRefused@test.com"
	user := newLoginUser(t, gormDB, login)
	tokenA, sessionA := loginSession(t, layout, login)
	tokenB, sessionB := loginSession(t, layout, login)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewAuthClient(conn)

	for _, req := range []*v1.LogoutRequest{
		{},
		// uid and realm no longer end the user's sessions
		{Uid: int32(user.ID), Realm: login},
	} {
		res, err := client.Logout(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int32(401), res.Code)
	}

	// no session was revoked
	for _, session := range []entities.Session{sessionA, sessionB} {
		assert.False(t, findSessionRow(t, gormDB, session.ID.String()).DeletedAt.Valid)
	}
	assert.Equal(t, 200, refreshOverHTTP(t, layout, tokenA).Code)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, tokenB).Code)
}
