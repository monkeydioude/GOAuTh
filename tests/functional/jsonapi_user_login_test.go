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

func TestJsonAPICanNotChangeLoginOnMissingUID(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/login", layout.Put(user.EditLogin))

	// login := "TestJsonAPICanNotChangeLoginOnMissingUID@test.com"
	passwd := "test"

	// create the user
	rec := httptest.NewRecorder()
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
	})
	assert.NoError(t, err)
	newLogin := "testtest@test.com"
	body, err := json.Marshal(entities.EditUserPayload{
		Password: passwd,
		NewLogin: &newLogin,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/user/login", bytes.NewReader(body))
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

func TestJsonAPICanNotChangeLoginOnIncorrectPassword(t *testing.T) {
	layout, _, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/login", layout.Put(user.EditLogin))

	// login := "TestJsonAPICanNotChangeLoginOnIncorrectPassword@test.com"
	// create the user
	rec := httptest.NewRecorder()
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID: 1,
	})
	assert.NoError(t, err)
	newLogin := "testtest@test.com"
	body, err := json.Marshal(entities.EditUserPayload{
		Password: "fake_pwd",
		NewLogin: &newLogin,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/user/login", bytes.NewReader(body))
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	assert.Equal(t, 400, rec.Code)
	body, err = io.ReadAll(rec.Body)
	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"Code":400,"Message":"InvalidCredentials"}`), body)
}

func TestJsonAPICanChangeAnUserLogin(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/login", layout.Put(user.EditLogin))

	login := "TestJsonAPICanChangeAnUserLogin@test.com"
	passwd := "testtt"
	newLogin := "new_login@test.com"
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
	}
	gormDB.Unscoped().Delete(&user, "login = ?", login)
	gormDB.Unscoped().Delete(&user, "login = ?", newLogin)
	defer gormDB.Unscoped().Delete(&user, "id = ?", user.ID)

	// create the user
	assert.Nil(t, gormDB.Create(&user).Error)
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID: user.ID,
	})
	assert.NoError(t, err)
	body, err := json.Marshal(entities.EditUserPayload{
		NewLogin: &newLogin,
		Password: passwd,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("PUT", "/user/login", bytes.NewReader(body))
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	userTrial := &entities.User{}
	assert.NoError(t, gormDB.Unscoped().First(userTrial, user.ID).Error)
	assert.Equal(t, 200, rec.Code)
	// signed version of "test"
	assert.Equal(t, newLogin, userTrial.Login)
	signedPasswd := crypt.HashPassword(passwd, layout.UserParams.Argon2params, layout.UserParams.PasswdSalt)
	assert.Equal(t, signedPasswd, userTrial.Password)
	body, err = io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.NoError(t, json.Unmarshal(body, &trial))
}
