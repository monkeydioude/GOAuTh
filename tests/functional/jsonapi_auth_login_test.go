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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanLogin(t *testing.T) {
	layout, gormDB, timeRef := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/login", layout.Put(auth.Login))
	rec := httptest.NewRecorder()
	login := "TestICanLogin@test.com"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)

	// define the user
	passwd := "test"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RealmName: login,
		RealmID:   realm.ID,
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})
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
	assert.Equal(t, 200, rec.Code)
	// retrieve the token from the response
	cookies, err := http.ParseCookie(rec.Result().Header["Set-Cookie"][0])
	assert.NoError(t, err)
	trialJWT, err := layout.JWTFactory.DecodeCookieToken(cookies[0])
	assert.NoError(t, err)
	// assert.Equal(t, trialJWT.Claims.Name, login)
	assert.Equal(t, trialJWT.Claims.Expire, int64(timeRef.Add(3*time.Second).Unix()))
	assert.Equal(t, trialJWT.Claims.Refresh, int64(timeRef.Add(10*time.Second).Unix()))
}
