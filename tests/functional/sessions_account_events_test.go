package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/user"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func editPasswordOverHTTP(t *testing.T, layout *handlers.Layout, accessToken string, password string, newPassword string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user/password", layout.Put(user.EditPassword))
	body, err := json.Marshal(entities.EditUserPayload{Password: password, NewPassword: &newPassword})
	assert.NoError(t, err)
	req := httptest.NewRequest("PUT", "/v1/user/password", bytes.NewReader(body))
	req.AddCookie(accessCookie(accessToken))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func assertRevoked(t *testing.T, layout *handlers.Layout, device deviceLogin, reason string) {
	stored := findSessionRow(t, layout.DB, device.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(reason), stored.RevokedReason)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, device.refreshToken).Code)
}

func TestPasswordChangeRevokesEverySessionButTheCallers(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestPasswordChangeRevokesEverySessionButTheCallers@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	tablet := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestPasswordChangeLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})

	// a refused change revokes nothing
	assert.Equal(t, 401, editPasswordOverHTTP(t, layout, laptop.accessToken, "wrong", "newpassword").Code)
	assert.False(t, findSessionRow(t, gormDB, phone.session.ID.String()).DeletedAt.Valid)

	assert.Equal(t, 200, editPasswordOverHTTP(t, layout, laptop.accessToken, "test", "newpassword").Code)

	assertRevoked(t, layout, phone, entities.SessionRevokedPasswordChanged)
	assertRevoked(t, layout, tablet, entities.SessionRevokedPasswordChanged)
	// the session that changed the password goes on
	assert.False(t, findSessionRow(t, gormDB, laptop.session.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, laptop.refreshToken).Code)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, other.refreshToken).Code)
}

func TestPasswordResetRevokesEverySession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestPasswordResetRevokesEverySession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestPasswordResetLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})

	created, err := services.UserActionCreate(gormDB, services.UserActionCreateIn{
		Login:  login,
		Realm:  login,
		Action: entities.UserActionTypePassword,
	}, uuid.NewString)
	assert.NoError(t, err)
	_, err = services.UserActionValidate(gormDB, layout.UserParams, services.UserActionValidateIn{
		Login:   login,
		Realm:   login,
		Data:    created.Data,
		Against: "newpassword",
	})
	assert.NoError(t, err)

	assertRevoked(t, layout, laptop, entities.SessionRevokedPasswordReset)
	assertRevoked(t, layout, phone, entities.SessionRevokedPasswordReset)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, other.refreshToken).Code)
}

func TestDeactivationRevokesEverySession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestDeactivationRevokesEverySession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestDeactivationLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user/deactivate", layout.Delete(user.Deactivate))
	req := httptest.NewRequest("DELETE", "/v1/user/deactivate", nil)
	req.AddCookie(accessCookie(laptop.accessToken))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	assertRevoked(t, layout, laptop, entities.SessionRevokedAccountDeactivated)
	assertRevoked(t, layout, phone, entities.SessionRevokedAccountDeactivated)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, other.refreshToken).Code)
}

func TestRPCDeleteRevokesEverySession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestRPCDeleteRevokesEverySession@test.com"
	deleted := newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestRPCDeleteLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})
	conn := setupRPC(t, layout)
	defer conn.Close()

	res, err := v1.NewAuthClient(conn).Delete(context.Background(), &v1.AuthIdRequest{Uid: int32(deleted.ID)})
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	assertRevoked(t, layout, laptop, entities.SessionRevokedAccountDeactivated)
	assertRevoked(t, layout, phone, entities.SessionRevokedAccountDeactivated)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, other.refreshToken).Code)
}
