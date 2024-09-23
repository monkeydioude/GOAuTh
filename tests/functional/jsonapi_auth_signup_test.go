package functional

import (
	"GOAuTh/internal/api/handlers/v1/auth"
	"GOAuTh/internal/domain/entities"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanSignup(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/signup", layout.Post(auth.Signup))
	login := "TestICanSignup@test.com"
	rec := httptest.NewRecorder()
	user := entities.User{
		Login:    login,
		Password: "test",
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)
	body, err := json.Marshal(user)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/v1/auth/signup", bytes.NewReader(body))
	assert.NoError(t, err)
	mux.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, 200)
	trial := entities.User{}
	assert.Nil(t, gormDB.Where("login = ?", login).First(&trial).Error)
	assert.Equal(t, trial.Login, login)
}
