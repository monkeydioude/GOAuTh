package functional

import (
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/domain/entities"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPCCanSignup(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	login := "TestRPCCanSignup@test.com"
	passwd := "test"

	conn := setupRPC(t, layout)
	defer conn.Close()

	client := v1.NewAuthClient(conn)
	ctx := context.Background()

	res, err := client.Signup(
		ctx,
		&v1.UserRequest{
			Login:    login,
			Password: passwd,
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)
	user := entities.User{}
	assert.NoError(t, json.Unmarshal([]byte(res.Message), &user))
	assert.Equal(t, uint(6), user.ID)
	assert.Nil(t, user.RevokedAt)
	assert.Equal(t, "TestRPCCanSignup@test.com", user.Login)
}
