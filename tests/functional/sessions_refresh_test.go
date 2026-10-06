package functional

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/jwt"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/ptr"
	v1 "github.com/monkeydioude/goauth/v2/pkg/grpc/v1"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

// loginSession logs the user in and returns the refresh token and the session it created.
func loginSession(t *testing.T, layout *handlers.Layout, login string) (string, entities.Session) {
	_, refreshCookie, err := services.AuthLogin(entities.NewUser(login, "test", login), services.ClientInfo{}, layout.DB, layout.UserParams, layout.AccessTokenFactory, layout.RefreshTokenFactory, layout.MaxActiveSessions)
	assert.NoError(t, err)
	refreshToken, err := layout.RefreshTokenFactory.DecodeToken(refreshCookie.Value)
	assert.NoError(t, err)
	return refreshCookie.Value, findSessionRow(t, layout.DB, refreshToken.Claims.SID)
}

// findSessionRow reads a session, revoked or not.
func findSessionRow(t *testing.T, gormDB *gorm.DB, sid string) entities.Session {
	var session entities.Session
	assert.NoError(t, gormDB.Unscoped().First(&session, "id = ?", sid).Error)
	return session
}

// refreshOverHTTP sends a refresh token to the JSON API, from a known client.
func refreshOverHTTP(t *testing.T, layout *handlers.Layout, refreshToken string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/refresh", layout.Put(jwt.Refresh))
	req, err := http.NewRequest("PUT", "/v1/jwt/refresh", nil)
	assert.NoError(t, err)
	req.RemoteAddr = "203.0.113.7:51234"
	req.Header.Set("User-Agent", "Mozilla/5.0 (refresh)")
	req.AddCookie(&http.Cookie{Name: consts.RefreshTokenCookie, Value: refreshToken})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// refreshedToken is the Refresh cookie a response set, "" when it set none.
func refreshedToken(rec *httptest.ResponseRecorder) string {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == consts.RefreshTokenCookie {
			return cookie.Value
		}
	}
	return ""
}

// errorMessage reads the message of a JSON API error response.
func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	var res response.HTTPResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	return res.Message
}

func refreshOverRPC(t *testing.T, client v1.JWTClient, in *v1.RefreshIn) (*v1.RefreshOut, metadata.MD) {
	var header metadata.MD
	out, err := client.Refresh(context.Background(), in, grpc.Header(&header))
	assert.NoError(t, err)
	return out, header
}

func TestJsonAPIRefreshRotatesOnlyItsSession(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPIRefreshRotatesOnlyItsSession@test.com"
	newLoginUser(t, gormDB, login)
	tokenA, sessionA := loginSession(t, layout, login)
	tokenB, sessionB := loginSession(t, layout, login)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}

	// several refreshes in a row, each with the token the previous one returned
	token := tokenA
	for i := 1; i <= 3; i++ {
		clock = timeRef.Add(time.Duration(i) * time.Minute)
		rec := refreshOverHTTP(t, layout, token)
		assert.Equal(t, 200, rec.Code)
		next := refreshedToken(rec)
		assert.NotEmpty(t, next)
		assert.NotEqual(t, token, next)
		token = next
	}

	// session A holds the last token, and knows the client and time of the last refresh
	rotated := findSessionRow(t, gormDB, sessionA.ID.String())
	assert.Equal(t, crypt.HashToken(token), rotated.TokenHash)
	assert.True(t, rotated.LastConnection.Equal(clock))
	assert.True(t, rotated.ExpiresAt.Equal(clock.Add(layout.RefreshTokenFactory.ExpiresIn)))
	assert.Equal(t, "203.0.113.7", rotated.LastIP)
	assert.Equal(t, "Mozilla/5.0 (refresh)", rotated.UserAgent)

	// session B was never touched, and still refreshes
	untouched := findSessionRow(t, gormDB, sessionB.ID.String())
	assert.Equal(t, sessionB.TokenHash, untouched.TokenHash)
	assert.True(t, sessionB.LastConnection.Equal(untouched.LastConnection))
	assert.True(t, sessionB.ExpiresAt.Equal(untouched.ExpiresAt))
	assert.Nil(t, untouched.RotatedAt)
	assert.Equal(t, 200, refreshOverHTTP(t, layout, tokenB).Code)
}

func TestJsonAPIRefreshWithinTheGraceWindow(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPIRefreshWithinTheGraceWindow@test.com"
	newLoginUser(t, gormDB, login)
	token, session := loginSession(t, layout, login)
	assert.Equal(t, 30*time.Second, layout.SessionReuseGrace)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}

	first := refreshOverHTTP(t, layout, token)
	assert.Equal(t, 200, first.Code)
	rotated := refreshedToken(first)

	// a refresh racing the first one with the same token, 10s later: an access token only
	clock = timeRef.Add(10 * time.Second)
	second := refreshOverHTTP(t, layout, token)
	assert.Equal(t, 200, second.Code)
	findCookie(t, second.Result().Cookies(), consts.AuthorizationCookie)
	assert.Equal(t, "", refreshedToken(second))

	// the session stays active, still holds the first refresh's token, which still works
	stored := findSessionRow(t, gormDB, session.ID.String())
	assert.False(t, stored.DeletedAt.Valid)
	assert.Equal(t, crypt.HashToken(rotated), stored.TokenHash)
	assert.True(t, stored.LastConnection.Equal(clock))
	assert.Equal(t, 200, refreshOverHTTP(t, layout, rotated).Code)
}

func TestJsonAPIConcurrentRefreshesRotateOnce(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	login := "TestJsonAPIConcurrentRefreshesRotateOnce@test.com"
	newLoginUser(t, gormDB, login)
	token, session := loginSession(t, layout, login)

	// five tabs refresh with the same token at once
	results := make(chan *httptest.ResponseRecorder, 5)
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- refreshOverHTTP(t, layout, token)
		}()
	}
	wg.Wait()
	close(results)

	// one rotates the token, the others get an access token only
	rotations := 0
	for rec := range results {
		assert.Equal(t, 200, rec.Code)
		if refreshedToken(rec) != "" {
			rotations++
		}
	}
	assert.Equal(t, 1, rotations)
	assert.False(t, findSessionRow(t, gormDB, session.ID.String()).DeletedAt.Valid)
}

func TestJsonAPIRefreshReuseAfterTheGraceWindowRevokesTheSession(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPIRefreshReuse@test.com"
	newLoginUser(t, gormDB, login)
	token, session := loginSession(t, layout, login)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}
	var logs bytes.Buffer
	defaultOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(defaultOutput) })

	rotated := refreshedToken(refreshOverHTTP(t, layout, token))
	assert.NotEmpty(t, rotated)

	// the old token comes back 31s later: it leaked, so its session is revoked
	clock = timeRef.Add(31 * time.Second)
	reuse := refreshOverHTTP(t, layout, token)
	assert.Equal(t, 401, reuse.Code)
	assert.Equal(t, consts.ERR_TOKEN_REUSED, errorMessage(t, reuse))
	stored := findSessionRow(t, gormDB, session.ID.String())
	assert.True(t, stored.DeletedAt.Valid)
	assert.Equal(t, entities.SessionRevokedReuseDetected, *stored.RevokedReason)

	// whoever holds the rotated token is logged out too
	revoked := refreshOverHTTP(t, layout, rotated)
	assert.Equal(t, 401, revoked.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, revoked))

	// the reuse is logged, without the tokens
	assert.Contains(t, logs.String(), "refresh token reused")
	assert.NotContains(t, logs.String(), token)
	assert.NotContains(t, logs.String(), rotated)
}

func TestJsonAPIRefreshRefusesExpiredAndRevokedSessions(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestJsonAPIRefreshRefuses@test.com"
	newLoginUser(t, gormDB, login)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}
	expiringToken, _ := loginSession(t, layout, login)
	revokedToken, revokedSession := loginSession(t, layout, login)
	assert.NoError(t, gormDB.Delete(&revokedSession).Error)

	revoked := refreshOverHTTP(t, layout, revokedToken)
	assert.Equal(t, 401, revoked.Code)
	assert.Equal(t, consts.ERR_TOKEN_REVOKED, errorMessage(t, revoked))

	// a token naming no session, as issued before sessions existed
	sessionless, err := layout.RefreshTokenFactory.GenerateToken(crypt.JWTDefaultClaims{UID: revokedSession.UserID, Realm: login})
	assert.NoError(t, err)
	missing := refreshOverHTTP(t, layout, sessionless.Token)
	assert.Equal(t, 401, missing.Code)
	assert.Equal(t, consts.ERR_TOKEN_MISSING_PARAMS, errorMessage(t, missing))

	// one second past its expiry, a session can't be refreshed anymore
	clock = timeRef.Add(layout.RefreshTokenFactory.ExpiresIn + time.Second)
	expired := refreshOverHTTP(t, layout, expiringToken)
	assert.Equal(t, 401, expired.Code)
	assert.Equal(t, consts.ERR_TOKEN_EXPIRED, errorMessage(t, expired))
}

func TestRPCRefreshRotatesAndStoresTheClient(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestRPCRefreshRotatesAndStoresTheClient@test.com"
	newLoginUser(t, gormDB, login)
	token, session := loginSession(t, layout, login)
	conn := setupRPC(t, layout)
	defer conn.Close()
	clock := timeRef.Add(time.Minute)
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}

	out, header := refreshOverRPC(t, v1.NewJWTClient(conn), &v1.RefreshIn{
		RefreshToken:            token,
		RefreshExpiresInSeconds: ptr.Ptr[int64](60),
		Client:                  &v1.ClientInfo{Ip: "198.51.100.1", UserAgent: "Mozilla/5.0 (rpc)"},
	})
	assert.NotEmpty(t, out.RefreshToken)
	assert.NotEqual(t, token, out.RefreshToken)
	// the refresh token lives as long as the session, not the 60s the caller asked for
	assert.Equal(t, clock.Add(layout.RefreshTokenFactory.ExpiresIn).Unix(), out.RefreshExpiresAt)
	cookie, err := rpc.FetchCookie(header, consts.RefreshTokenCookie)
	assert.NoError(t, err)
	assert.Equal(t, out.RefreshToken, cookie.Value)

	stored := findSessionRow(t, gormDB, session.ID.String())
	assert.Equal(t, crypt.HashToken(out.RefreshToken), stored.TokenHash)
	assert.Equal(t, "198.51.100.1", stored.LastIP)
	assert.Equal(t, "Mozilla/5.0 (rpc)", stored.UserAgent)
}

func TestRPCRefreshWithinTheGraceWindowReturnsNoRefreshToken(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestRPCRefreshWithinTheGraceWindow@test.com"
	newLoginUser(t, gormDB, login)
	token, _ := loginSession(t, layout, login)
	conn := setupRPC(t, layout)
	defer conn.Close()
	client := v1.NewJWTClient(conn)
	clock := timeRef
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return clock
	}

	first, _ := refreshOverRPC(t, client, &v1.RefreshIn{RefreshToken: token})
	assert.NotEmpty(t, first.RefreshToken)

	// a refresh racing the first one with the same token, 10s later
	clock = timeRef.Add(10 * time.Second)
	second, header := refreshOverRPC(t, client, &v1.RefreshIn{RefreshToken: token})
	assert.NotEmpty(t, second.AccessToken)
	assert.Equal(t, "", second.RefreshToken)
	// the session expiry the first refresh set
	assert.Equal(t, first.RefreshExpiresAt, second.RefreshExpiresAt)
	_, err := rpc.FetchCookie(header, consts.RefreshTokenCookie)
	assert.Error(t, err)
}
