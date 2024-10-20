package functional

import (
	"GOAuTh/internal/api/handlers/v1/jwt"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanRefreshAValidToken(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))

	login := "TestICanRefreshAValidToken@test.com"
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		ID:        1,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		UID: 1,
	})
	assert.NoError(t, err)
	timeRef := layout.JWTFactory.TimeFn()
	// we go 5s forward in time, so we are in refresh spot
	// (expiresIn 3s, refreshesIn 10s, 5 is between, so it's cool)
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef.Add(5 * time.Second)
	}
	req, err := http.NewRequest("POST", "/jwt/refresh", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.NoError(t, json.Unmarshal(body, &trial))
	strings.Split(trial.Value, " ")
	jwt2, err := layout.JWTFactory.DecodeCookieToken(&trial)
	assert.NoError(t, err)
	// 5 + 3 seconds, because we went 5s forward in time, and JWTFactory ExpiresIn config is 3s
	assert.NotEqual(t, jwt2.Token, jwt.Token)
	assert.Equal(t, jwt2.Claims.Expire, timeRef.Add((5+3)*time.Second).Unix())
}

func TestJsonAPIGetA401OnRefreshingAnInvalidToken(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second
	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))
	// login := "TestIGetA401OnRefreshingAnInvalidToken@test.com"
	rec := httptest.NewRecorder()
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
	})
	assert.NoError(t, err)
	timeRef := layout.JWTFactory.TimeFn()
	// we go 12s forward in time, so we are beyond refresh time.
	// RefreshesIn 10s when we generated thee token
	layout.JWTFactory.TimeFn = func() time.Time {
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
