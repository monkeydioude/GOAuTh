package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/domain/entities"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRPCCanSignup(t *testing.T) {
	layout, gormDB, _ := setup()
	login := "TestRPCCanSignup@test.com"
	passwd := "test"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         "test1",
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	conn := setupRPC(t, layout)
	t.Cleanup(func() {
		conn.Close()
		cleanup(layout)
		gormDB.Unscoped().Delete(&realm)
	})

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	res, err := client.Signup(
		ctx,
		&v1.UserRequest{
			Login:    login,
			Password: passwd,
			Realm:    realm.Name,
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	user := entities.User{}
	assert.NoError(t, json.Unmarshal([]byte(res.Message), &user))
	assert.Equal(t, realm.ID, user.RealmID)
	assert.Nil(t, user.RevokedAt)
	assert.Equal(t, "TestRPCCanSignup@test.com", user.Login)
}
