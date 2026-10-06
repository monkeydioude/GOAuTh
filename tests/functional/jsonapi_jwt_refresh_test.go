package functional

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers/v1/jwt"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanRefreshAValidToken(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	login := "TestICanRefreshAValidToken@test.com"
	newLoginUser(t, gormDB, login)
	refreshToken, _ := loginSession(t, layout, login)

	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))
	// 5s after the login
	layout.RefreshTokenFactory.TimeFn = func() time.Time {
		return timeRef.Add(5 * time.Second)
	}
	req, err := http.NewRequest("POST", "/jwt/refresh", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  consts.RefreshTokenCookie,
		Value: refreshToken,
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	refreshed, err := layout.RefreshTokenFactory.DecodeToken(findCookie(t, rec.Result().Cookies(), consts.RefreshTokenCookie).Value)
	assert.NoError(t, err)
	assert.NotEqual(t, refreshToken, refreshed.Token)
	// the session lives for its whole TTL again, counted from this refresh
	assert.Equal(t, timeRef.Add(5*time.Second).Add(layout.RefreshTokenFactory.ExpiresIn).Unix(), refreshed.Claims.Expire)
}

func TestJsonAPIGetA401OnRefreshingAnInvalidToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.AccessTokenFactory.ExpiresIn = 3 * time.Second
	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))
	// login := "TestIGetA401OnRefreshingAnInvalidToken@test.com"
	rec := httptest.NewRecorder()
	jwt, err := layout.AccessTokenFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
	})
	assert.NoError(t, err)
	timeRef := layout.AccessTokenFactory.TimeFn()
	// we go 12s forward in time, so we are beyond refresh time.
	// RefreshesIn 10s when we generated thee token
	layout.AccessTokenFactory.TimeFn = func() time.Time {
		return timeRef.Add(12 * time.Second)
	}
	req, err := http.NewRequest("POST", "/jwt/refresh", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	// should fail
	assert.Equal(t, rec.Code, 401)
}
