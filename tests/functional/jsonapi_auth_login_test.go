package functional

import (
	"GOAuTh/internal/api/handlers/v1/auth"
	"GOAuTh/internal/domain/entities"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanLogin(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", layout.Put(auth.Login))
	rec := httptest.NewRecorder()
	// define the user
	login := "TestICanLogin@test.com"
	passwd := "test"
	user := entities.User{
		Login:    login,
		Password: passwd,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)
	// create the user
	assert.Nil(t, gormDB.Create(&user).Error)
	// restore password to its previous state, before it got modified by gorm
	user.Password = passwd
	// prepare to request the endpoint
	body, err := json.Marshal(user)
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/v1/auth/login", bytes.NewReader(body))
	assert.NoError(t, err)
	mux.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, 200)
	// retrieve the token from the response
	cookies, err := http.ParseCookie(rec.Result().Header["Set-Cookie"][0])
	assert.NoError(t, err)
	responseToken := cookies[0].Value
	trialJWT, err := layout.JWTFactory.DecodeToken(responseToken)
	assert.NoError(t, err)
	assert.Equal(t, trialJWT.Claims.Name, login)
	assert.Equal(t, trialJWT.Claims.Expire, int64(timeRef.Add(3*time.Second).Unix()))
	assert.Equal(t, trialJWT.Claims.Refresh, int64(timeRef.Add(10*time.Second).Unix()))
}
