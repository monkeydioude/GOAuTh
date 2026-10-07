package functional

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/auth"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/session"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"

	"github.com/stretchr/testify/assert"
)

// sessionsOverHTTP sends a request to the session and logout routes, with the given cookies.
func sessionsOverHTTP(t *testing.T, layout *handlers.Layout, method string, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/sessions", layout.Get(session.List))
	mux.HandleFunc("DELETE /v1/sessions", layout.Delete(session.RevokeAll))
	mux.HandleFunc("DELETE /v1/sessions/{id}", layout.Delete(session.Revoke))
	mux.HandleFunc("PUT /v1/auth/logout", layout.Put(auth.Logout))
	req := httptest.NewRequest(method, target, nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func accessCookie(accessToken string) *http.Cookie {
	return &http.Cookie{Name: consts.AuthorizationCookie, Value: "Bearer " + accessToken}
}

func refreshCookie(refreshToken string) *http.Cookie {
	return &http.Cookie{Name: consts.RefreshTokenCookie, Value: refreshToken}
}

func listedSessions(t *testing.T, rec *httptest.ResponseRecorder) []session.SessionOut {
	var out session.ListOut
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	return out.Sessions
}

func TestHTTPSessionListShowsTheCallersSessions(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPSessionListShowsTheCallersSessions@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.NewClientInfo("203.0.113.7", "Mozilla/5.0 (laptop)"))
	phone := loginDevice(t, layout, login, services.NewClientInfo("198.51.100.4", "Mozilla/5.0 (phone)"))
	revoked := loginDevice(t, layout, login, services.ClientInfo{})
	expired := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestHTTPSessionListLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	loginDevice(t, layout, otherLogin, services.ClientInfo{})
	now := layout.AccessTokenFactory.TimeFn()
	setLastConnection(t, layout, laptop.session.ID.String(), now.Add(-2*time.Hour))
	setLastConnection(t, layout, phone.session.ID.String(), now.Add(-1*time.Hour))
	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, now.Add(-30*time.Minute), "id = ?", revoked.session.ID))
	assert.NoError(t, gormDB.Model(&entities.Session{}).Where("id = ?", expired.session.ID).Update("expires_at", now.Add(-time.Second)).Error)

	// active sessions only, most recently used first
	rec := sessionsOverHTTP(t, layout, "GET", "/v1/sessions", accessCookie(laptop.accessToken))
	assert.Equal(t, 200, rec.Code)
	sessions := listedSessions(t, rec)
	assert.Len(t, sessions, 2)
	assert.Equal(t, phone.session.ID.String(), sessions[0].ID)
	assert.False(t, sessions[0].Current)
	assert.Equal(t, "198.51.100.4", sessions[0].LoginIP)
	assert.Equal(t, "Mozilla/5.0 (phone)", sessions[0].UserAgent)
	assert.Equal(t, laptop.session.ID.String(), sessions[1].ID)
	assert.True(t, sessions[1].Current)
	assert.Equal(t, "203.0.113.7", sessions[1].LastIP)
	assert.Equal(t, now.Add(-2*time.Hour).Unix(), sessions[1].LastConnection.Unix())
	assert.Equal(t, laptop.session.ExpiresAt.Unix(), sessions[1].ExpiresAt.Unix())
	assert.Nil(t, sessions[1].RevokedAt)
	assert.NotContains(t, rec.Body.String(), "revoked_at")

	// include_revoked adds the revoked and the expired session
	rec = sessionsOverHTTP(t, layout, "GET", "/v1/sessions?include_revoked=true", accessCookie(laptop.accessToken))
	assert.Equal(t, 200, rec.Code)
	sessions = listedSessions(t, rec)
	assert.Len(t, sessions, 4)
	byID := map[string]session.SessionOut{}
	for _, info := range sessions {
		byID[info.ID] = info
	}
	assert.Equal(t, now.Add(-30*time.Minute).Unix(), byID[revoked.session.ID.String()].RevokedAt.Unix())
	assert.Equal(t, entities.SessionRevokedByUser, byID[revoked.session.ID.String()].RevokedReason)
	assert.Nil(t, byID[expired.session.ID.String()].RevokedAt)
}

func TestHTTPSessionRoutesNeedAnActiveAccessToken(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPSessionRoutesNeedAnActiveAccessToken@test.com"
	newLoginUser(t, gormDB, login)
	device := loginDevice(t, layout, login, services.ClientInfo{})
	revoked := loginDevice(t, layout, login, services.ClientInfo{})
	assert.NoError(t, services.RevokeSessions(gormDB, entities.SessionRevokedByUser, layout.AccessTokenFactory.TimeFn(), "id = ?", revoked.session.ID))

	// no access token, even with a refresh token
	rec := sessionsOverHTTP(t, layout, "GET", "/v1/sessions", refreshCookie(device.refreshToken))
	assert.Equal(t, 401, rec.Code)

	// a refresh token is not an access token
	rec = sessionsOverHTTP(t, layout, "GET", "/v1/sessions", accessCookie(device.refreshToken))
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_WRONG_TOKEN_TYPE, errorMessage(t, rec))

	// the access token of a revoked session
	rec = sessionsOverHTTP(t, layout, "GET", "/v1/sessions", accessCookie(revoked.accessToken))
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, rec))

	rec = sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions/"+device.session.ID.String(), accessCookie(revoked.accessToken))
	assert.Equal(t, 401, rec.Code)
	rec = sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions", accessCookie(revoked.accessToken))
	assert.Equal(t, 401, rec.Code)
	assert.False(t, findSessionRow(t, gormDB, device.session.ID.String()).DeletedAt.Valid)
}

func TestHTTPSessionRevokeEndsOneSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPSessionRevokeEndsOneSession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestHTTPSessionRevokeLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})

	rec := sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions/"+phone.session.ID.String(), accessCookie(laptop.accessToken))
	assert.Equal(t, 200, rec.Code)

	stored := findSessionRow(t, gormDB, phone.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedByUser), stored.RevokedReason)
	assert.Equal(t, 401, statusOverHTTP(t, layout, phone.accessToken).Code)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, phone.refreshToken).Code)
	// the caller's session goes on
	assert.Equal(t, 200, statusOverHTTP(t, layout, laptop.accessToken).Code)

	// already revoked
	rec = sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions/"+phone.session.ID.String(), accessCookie(laptop.accessToken))
	assert.Equal(t, 404, rec.Code)
	assert.Equal(t, consts.ERR_SESSION_NOT_FOUND, errorMessage(t, rec))

	// another user's session is not found, and left alone
	rec = sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions/"+other.session.ID.String(), accessCookie(laptop.accessToken))
	assert.Equal(t, 404, rec.Code)
	assert.False(t, findSessionRow(t, gormDB, other.session.ID.String()).DeletedAt.Valid)

	rec = sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions/not-a-session-id", accessCookie(laptop.accessToken))
	assert.Equal(t, 404, rec.Code)
}

func TestHTTPSessionRevokeAllCanKeepTheCurrentSession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPSessionRevokeAllCanKeepTheCurrentSession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	tablet := loginDevice(t, layout, login, services.ClientInfo{})
	otherLogin := "TestHTTPSessionRevokeAllLeavesOtherUsers@test.com"
	newLoginUser(t, gormDB, otherLogin)
	other := loginDevice(t, layout, otherLogin, services.ClientInfo{})

	rec := sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions?keep_current=true", accessCookie(laptop.accessToken))
	assert.Equal(t, 200, rec.Code)

	for _, device := range []deviceLogin{phone, tablet} {
		stored := findSessionRow(t, gormDB, device.session.ID.String())
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogoutAll), stored.RevokedReason)
		assert.Equal(t, 401, refreshOverHTTP(t, layout, device.refreshToken).Code)
	}
	assert.False(t, findSessionRow(t, gormDB, laptop.session.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, statusOverHTTP(t, layout, laptop.accessToken).Code)
	assert.False(t, findSessionRow(t, gormDB, other.session.ID.String()).DeletedAt.Valid)
}

func TestHTTPSessionRevokeAllEndsEverySession(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPSessionRevokeAllEndsEverySession@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})

	rec := sessionsOverHTTP(t, layout, "DELETE", "/v1/sessions", accessCookie(laptop.accessToken))
	assert.Equal(t, 200, rec.Code)

	for _, device := range []deviceLogin{laptop, phone} {
		stored := findSessionRow(t, gormDB, device.session.ID.String())
		assert.True(t, stored.DeletedAt.Valid)
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogoutAll), stored.RevokedReason)
	}
	assert.Equal(t, 401, statusOverHTTP(t, layout, laptop.accessToken).Code)
}

func TestHTTPLogoutEndsTheCallersSessionAndClearsTheCookies(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPLogoutEndsTheCallersSessionAndClearsTheCookies@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})

	// the Refresh cookie names the session, over the access token of another one
	rec := sessionsOverHTTP(t, layout, "PUT", "/v1/auth/logout", refreshCookie(laptop.refreshToken), accessCookie(phone.accessToken))
	assert.Equal(t, 200, rec.Code)

	stored := findSessionRow(t, gormDB, laptop.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), stored.RevokedReason)
	assert.Equal(t, 401, refreshOverHTTP(t, layout, laptop.refreshToken).Code)
	for _, name := range []string{consts.AuthorizationCookie, consts.RefreshTokenCookie} {
		cleared := findCookie(t, rec.Result().Cookies(), name)
		assert.Empty(t, cleared.Value)
		assert.Less(t, cleared.MaxAge, 0)
		assert.Equal(t, "/", cleared.Path)
	}
	// the user's other session goes on
	assert.False(t, findSessionRow(t, gormDB, phone.session.ID.String()).DeletedAt.Valid)
	assert.Equal(t, 200, statusOverHTTP(t, layout, phone.accessToken).Code)

	// logging out of a session already ended is a no-op
	rec = sessionsOverHTTP(t, layout, "PUT", "/v1/auth/logout", refreshCookie(laptop.refreshToken))
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), findSessionRow(t, gormDB, laptop.session.ID.String()).RevokedReason)
}

func TestHTTPLogoutWorksWithAnExpiredAccessToken(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestHTTPLogoutWorksWithAnExpiredAccessToken@test.com"
	newLoginUser(t, gormDB, login)
	laptop := loginDevice(t, layout, login, services.ClientInfo{})
	phone := loginDevice(t, layout, login, services.ClientInfo{})
	expiredAccess, err := layout.AccessTokenFactory.WithExpiresIn(-time.Minute).GenerateToken(crypt.JWTDefaultClaims{
		UID:   laptop.session.UserID,
		Realm: login,
		SID:   laptop.session.ID.String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, 401, statusOverHTTP(t, layout, expiredAccess.GetToken()).Code)

	// no cookie at all
	rec := sessionsOverHTTP(t, layout, "PUT", "/v1/auth/logout")
	assert.Equal(t, 401, rec.Code)

	// an access token is not a refresh token
	rec = sessionsOverHTTP(t, layout, "PUT", "/v1/auth/logout", refreshCookie(phone.accessToken))
	assert.Equal(t, 401, rec.Code)
	assert.False(t, findSessionRow(t, gormDB, phone.session.ID.String()).DeletedAt.Valid)

	rec = sessionsOverHTTP(t, layout, "PUT", "/v1/auth/logout", accessCookie(expiredAccess.GetToken()))
	assert.Equal(t, 200, rec.Code)
	stored := findSessionRow(t, gormDB, laptop.session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, ptr.Ptr(entities.SessionRevokedLogout), stored.RevokedReason)
	assert.False(t, findSessionRow(t, gormDB, phone.session.ID.String()).DeletedAt.Valid)
}
