package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/auth"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

func newLoginUser(t *testing.T, gormDB *gorm.DB, login string) entities.User {
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  "test",
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	assert.NoError(t, gormDB.Create(&user).Error)
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})
	return user
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	idx := slices.IndexFunc(cookies, func(c *http.Cookie) bool {
		return c.Name == name
	})
	if idx == -1 {
		t.Fatalf("no %s cookie in the response", name)
	}
	return cookies[idx]
}

func TestJsonAPILoginCreatesOneSessionPerLogin(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPILoginCreatesOneSessionPerLogin@test.com"
	user := newLoginUser(t, gormDB, login)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", layout.Put(auth.Login))

	sids := map[string]bool{}
	for range 2 {
		body, err := json.Marshal(auth.LoginIn{Login: login, Password: "test", RealmName: login})
		assert.NoError(t, err)
		req, err := http.NewRequest("PUT", "/v1/auth/login", bytes.NewReader(body))
		assert.NoError(t, err)
		req.RemoteAddr = "203.0.113.7:51234"
		req.Header.Set("User-Agent", "Mozilla/5.0 (test)")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assert.Equal(t, 200, rec.Code)

		accessToken, err := layout.AccessTokenFactory.DecodeCookieToken(findCookie(t, rec.Result().Cookies(), consts.AuthorizationCookie))
		assert.NoError(t, err)
		refreshCookie := findCookie(t, rec.Result().Cookies(), consts.RefreshTokenCookie)
		refreshToken, err := layout.RefreshTokenFactory.DecodeToken(refreshCookie.Value)
		assert.NoError(t, err)
		// both tokens name the session created by this login
		assert.NotEmpty(t, refreshToken.Claims.SID)
		assert.Equal(t, refreshToken.Claims.SID, accessToken.Claims.SID)
		sids[refreshToken.Claims.SID] = true

		var session entities.Session
		assert.NoError(t, gormDB.First(&session, "id = ?", refreshToken.Claims.SID).Error)
		assert.Equal(t, user.ID, session.UserID)
		assert.Equal(t, crypt.HashToken(refreshCookie.Value), session.TokenHash)
		assert.Equal(t, "203.0.113.7", session.LoginIP)
		assert.Equal(t, "203.0.113.7", session.LastIP)
		assert.Equal(t, "Mozilla/5.0 (test)", session.UserAgent)
		assert.True(t, session.LastConnection.Equal(timeRef))
		assert.True(t, session.ExpiresAt.Equal(time.Unix(refreshToken.Claims.Expire, 0)))
	}
	assert.Len(t, sids, 2)

	var stored entities.User
	assert.NoError(t, gormDB.First(&stored, user.ID).Error)
	// only the hash is stored: the old single-token column stays empty
	assert.Nil(t, stored.RefreshToken)
	if assert.NotNil(t, stored.LastLoggedAt) {
		assert.True(t, stored.LastLoggedAt.Equal(timeRef))
	}
}

func TestRPCLoginStoresTheClientAndIgnoresTheRefreshTTL(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestRPCLoginStoresTheClient@test.com"
	user := newLoginUser(t, gormDB, login)
	conn := setupRPC(t, layout)
	defer conn.Close()

	var headerMD metadata.MD
	res, err := v1.NewAuthClient(conn).Login(context.Background(), &v1.UserRequest{
		Login:                   login,
		Password:                "test",
		Realm:                   login,
		RefreshExpiresInSeconds: ptr.Ptr[int64](60),
		Client:                  &v1.ClientInfo{Ip: "198.51.100.1", UserAgent: "Mozilla/5.0 (rpc)"},
	}, grpc.Header(&headerMD))
	assert.NoError(t, err)
	assert.Equal(t, int32(200), res.Code)

	// the refresh token lives as long as the session, not the 60s the caller asked for
	refreshCookie, err := rpc.FetchCookie(headerMD, consts.RefreshTokenCookie)
	assert.NoError(t, err)
	assert.Equal(t, int(layout.RefreshTokenFactory.ExpiresIn.Seconds()), refreshCookie.MaxAge)

	var session entities.Session
	assert.NoError(t, gormDB.First(&session, "user_id = ?", user.ID).Error)
	assert.Equal(t, "198.51.100.1", session.LoginIP)
	assert.Equal(t, "198.51.100.1", session.LastIP)
	assert.Equal(t, "Mozilla/5.0 (rpc)", session.UserAgent)
	assert.True(t, session.ExpiresAt.Equal(timeRef.Add(layout.RefreshTokenFactory.ExpiresIn)))
}

func TestLoginBeyondTheCapRevokesTheLeastRecentlyUsedSession(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestLoginBeyondTheCap@test.com"
	user := newLoginUser(t, gormDB, login)
	assert.Equal(t, 10, layout.MaxActiveSessions)

	// one login per minute, so the first one is the least recently used
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}
	for i := range 11 {
		clock = timeRef.Add(time.Duration(i) * time.Minute)
		_, _, err := services.AuthLogin(entities.NewUser(login, "test", login), services.ClientInfo{}, gormDB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
		assert.NoError(t, err)
	}

	var active []entities.Session
	assert.NoError(t, gormDB.Where("user_id = ?", user.ID).Find(&active).Error)
	assert.Len(t, active, 10)
	var revoked []entities.Session
	assert.NoError(t, gormDB.Unscoped().Where("user_id = ? AND deleted_at IS NOT NULL", user.ID).Find(&revoked).Error)
	if assert.Len(t, revoked, 1) {
		assert.True(t, revoked[0].LastConnection.Equal(timeRef))
		assert.Equal(t, ptr.Ptr(entities.SessionRevokedLimitExceeded), revoked[0].RevokedReason)
	}
}
