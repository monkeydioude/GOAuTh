package functional

import (
	"GOAuTh/internal/api/handlers/v1/jwt"
	"GOAuTh/pkg/crypt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
		Name: login,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, 200)
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.Equal(t, json.Unmarshal(body, &trial), nil)
	assert.Equal(t, trial.Value, jwt.Token)
}

func TestJsonAPIGetA401OnInvalidToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/jwt/status", layout.Post(jwt.Status))

	login := "TestIGetA401OnInvalidToken@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	assert.NoError(t, err)
	// assign an invalid value
	jwt.Token = "bs-value"
	req, err := http.NewRequest("POST", "/v1/jwt/status", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	// should fail
	assert.Equal(t, rec.Code, 401)
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.Equal(t, json.Unmarshal(body, &trial), nil)
	assert.NotEqual(t, trial.Value, jwt.Token)
}
