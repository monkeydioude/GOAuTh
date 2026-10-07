package functional

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/user"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// loginAccessJWT logs the user of realm login in with password: the access token of a new, active session.
func loginAccessJWT(t *testing.T, layout *handlers.Layout, login string, password string) entities.JWT[crypt.JWTDefaultClaims] {
	accessCookie, _, err := services.AuthLogin(entities.NewUser(login, password, login), services.ClientInfo{}, layout.DB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
	assert.NoError(t, err)
	jwt, err := layout.AccessTokenFactory.DecodeCookieToken(&accessCookie)
	assert.NoError(t, err)
	return jwt
}

func editLoginOverHTTP(t *testing.T, layout *handlers.Layout, accessToken string, password string, newLogin string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user/login", layout.Put(user.EditLogin))
	body, err := json.Marshal(entities.EditUserPayload{Password: password, NewLogin: &newLogin})
	assert.NoError(t, err)
	req := httptest.NewRequest("PUT", "/v1/user/login", bytes.NewReader(body))
	req.AddCookie(accessCookie(accessToken))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func deactivateOverHTTP(t *testing.T, layout *handlers.Layout, accessToken string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/user/deactivate", layout.Delete(user.Deactivate))
	req := httptest.NewRequest("DELETE", "/v1/user/deactivate", nil)
	req.AddCookie(accessCookie(accessToken))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func storedUser(t *testing.T, gormDB *gorm.DB, uid uint) entities.User {
	var stored entities.User
	assert.NoError(t, gormDB.Unscoped().First(&stored, uid).Error)
	return stored
}

func TestAccountEndpointsRefuseARevokedSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestAccountEndpointsRefuseARevokedSession@test.com"
	registered := newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, layout.AccessTokenFactory.TimeFn(), "id = ?", phone.session.ID))
	before := storedUser(t, gormDB, registered.ID)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewUserClient(conn)

	// even with the right password
	rec := editPasswordOverHTTP(t, layout, phone.accessToken, "test", "newpassword")
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))
	rec = editLoginOverHTTP(t, layout, phone.accessToken, "test", "new-"+login)
	assert.Equal(t, 401, rec.Code)
	rec = deactivateOverHTTP(t, layout, phone.accessToken)
	assert.Equal(t, 401, rec.Code)
	res, err := client.EditUser(withAccessToken(phone.accessToken), &v1.EditUserRequest{Password: "test", NewPassword: "newpassword"})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	res, err = client.Deactivate(withAccessToken(phone.accessToken), &v1.Empty{})
	assert.NoError(t, err)
	assert.Equal(t, int32(401), res.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, res.Message)

	// the account is untouched, and its active session goes on
	after := storedUser(t, gormDB, registered.ID)
	assert.Equal(t, before.Password, after.Password)
	assert.Equal(t, login, after.Login)
	assert.False(t, after.DeletedAt.Valid)
	assert.Equal(t, 200, statusOverHTTP(t, layout, laptop.accessToken).Code)
}

func TestAccountEndpointsRefuseAnExpiredAccessToken(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestAccountEndpointsRefuseAnExpiredAccessToken@test.com"
	registered := newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	// its session is still active: only the access token is expired
	expired, err := layout.AccessTokenFactory.WithExpiresIn(-time.Minute).GenerateToken(crypt.JWTDefaultClaims{
		UID:   registered.ID,
		Realm: login,
		SID:   laptop.session.ID.String(),
	})
	assert.NoError(t, err)
	before := storedUser(t, gormDB, registered.ID)

	rec := editPasswordOverHTTP(t, layout, expired.GetToken(), "test", "newpassword")
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_EXPIRED, errorMessage(t, rec))
	rec = editLoginOverHTTP(t, layout, expired.GetToken(), "test", "new-"+login)
	assert.Equal(t, 401, rec.Code)
	rec = deactivateOverHTTP(t, layout, expired.GetToken())
	assert.Equal(t, 401, rec.Code)

	after := storedUser(t, gormDB, registered.ID)
	assert.Equal(t, before.Password, after.Password)
	assert.Equal(t, login, after.Login)
	assert.False(t, after.DeletedAt.Valid)
	// the session's own access token still works
	assert.Equal(t, 200, editPasswordOverHTTP(t, layout, laptop.accessToken, "test", "newpassword").Code)
}
