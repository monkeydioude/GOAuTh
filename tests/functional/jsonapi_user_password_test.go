package functional

import (
	"GOAuTh/internal/api/handlers/v1/user"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanNotChangePasswordOnMissingUID(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/password", layout.Put(user.EditPassword))

	login := "TestJsonAPICanNotChangePasswordOnMissingUID@test.com"
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
		// Name: login,
	})
	assert.NoError(t, err)
	newPasswd := "testtest"
	body, err := json.Marshal(entities.EditUserPayload{
		Password:    passwd,
		NewPassword: &newPasswd,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/user/password", bytes.NewReader(body))
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	assert.Equal(t, 401, rec.Code)
	body, err = io.ReadAll(rec.Body)
	assert.NoError(t, err)
	assert.NotEqual(t, []byte(`{"Code":401,"Message":"Unauthorized"}`), body)
}

func TestJsonAPICanChangeAnUserPassword(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/password", layout.Put(user.EditPassword))

	login := "TestJsonAPICanChangeAnUserPassword@test.com"
	passwd := "testtt"
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
		// Name: login,
		UID: 1,
	})
	assert.NoError(t, err)
	newPasswd := "test"
	body, err := json.Marshal(entities.EditUserPayload{
		Password:    passwd,
		NewPassword: &newPasswd,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/user/password", bytes.NewReader(body))
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	userTrial := &entities.User{}
	assert.NoError(t, gormDB.Unscoped().First(userTrial, 1).Error)
	assert.Equal(t, 200, rec.Code)
	// signed version of "test"
	assert.Equal(t, "TJQG9V6gx0aVGZBSa3sTkHtmbD1fAngcrA4SxhndGXY=", userTrial.Password)
	body, err = io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.NoError(t, json.Unmarshal(body, &trial))
}
