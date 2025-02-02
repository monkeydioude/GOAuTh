package functional

import (
	"GOAuTh/internal/api/handlers/v1/jwt"
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/http/response"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanGetAValidTokensStatus(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/status", layout.Post(jwt.Status))

	login := "TestICanGetAValidTokensStatus@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		UID:   1,
		Realm: login,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	trial, err := http.ParseCookie(rec.Header().Get("Set-Cookie"))
	assert.NoError(t, err)
	assert.Equal(t, "Bearer "+jwt.Token, trial[0].Value)
}

func TestJsonAPIGetA401OnInvalidToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/status", layout.Post(jwt.Status))

	// login := "TestIGetA401OnInvalidToken@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
	})
	assert.NoError(t, err)
	// assign an invalid value
	jwt.Token = "bs-value"
	req, err := http.NewRequest("POST", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	// should fail
	assert.Equal(t, 401, rec.Code)
	trial, err := http.ParseCookie(rec.Header().Get("Set-Cookie"))
	assert.Empty(t, trial)
	assert.Error(t, err)
}

func TestJsonAPIGetA401OnExpiredToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/status", layout.Post(jwt.Status))

	// login := "TestIGetA401OnInvalidToken@test.com"
	rec := httptest.NewRecorder()
	// ensuring factory's timeline
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		UID:   1,
		Realm: "cabane123",
		// Name: login,
	})
	assert.NoError(t, err)
	timeRef := layout.JWTFactory.TimeFn()
	// sending the factory into the futur, making sure we are past refresh time
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef.Add(12 * time.Second)
	}
	req, err := http.NewRequest("POST", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	// should fail
	assert.Equal(t, 401, rec.Code)
	trial, err := http.ParseCookie(rec.Header().Get("Set-Cookie"))
	assert.Empty(t, trial)
	assert.Error(t, err)
	trialRes := response.HTTPResponse{}
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	assert.Equal(t, nil, json.Unmarshal(body, &trialRes))
	assert.Equal(t, consts.ERR_TOKEN_EXPIRED, trialRes.Message)
}
