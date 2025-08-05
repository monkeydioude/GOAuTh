package functional

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monkeydioude/goauth/internal/api/handlers/v1/auth"
	"github.com/monkeydioude/goauth/internal/domain/entities"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanSignup(t *testing.T) {
	layout, gormDB, _ := setup()
	defer cleanup(layout)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/auth/signup", layout.Post(auth.Signup))
	login := "TestICanSignup@test.com"
	rec := httptest.NewRecorder()
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         "test1",
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  "test",
		RealmID:   realm.ID,
		RealmName: "test1",
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})
	body, err := json.Marshal(user)
	assert.NoError(t, err)
	req, err := http.NewRequest("POST", "/v1/auth/signup", bytes.NewReader(body))
	assert.NoError(t, err)
	mux.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, 200)
	trial := entities.User{}
	assert.Nil(t, gormDB.Where("login = ?", login).First(&trial).Error)
	assert.Equal(t, login, trial.Login)
}
