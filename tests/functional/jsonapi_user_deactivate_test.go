package functional

import (
	"GOAuTh/internal/api/handlers/v1/user"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJsonAPICanNotDeactivateAJWTMissingUID(t *testing.T) {
	layout, gormDB, _ := setup()
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/deactivate", layout.Delete(user.Deactivate))

	login := "TestJsonAPICanNotDeactivateAJWTMissingUID@test.com"
	passwd := "test"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		ID:        1,
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	t.Cleanup(func() {
		cleanup(layout)
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})
	// create the user
	assert.Nil(t, gormDB.Save(&user).Error)
	rec := httptest.NewRecorder()
	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		// UID:   user.ID,
		Realm: realm.Name,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("DELETE", "/user/deactivate", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	user = entities.User{}
	assert.NoError(t, gormDB.Unscoped().First(&user, 1).Error)
	assert.Equal(t, 401, rec.Code)
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	assert.NotEqual(t, []byte("Ok"), body)
	assert.False(t, user.DeletedAt.Valid)
}

func TestJsonAPICanDeactivateAnUserByID(t *testing.T) {
	layout, gormDB, _ := setup()
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/user/deactivate", layout.Delete(user.Deactivate))

	login := "TestJsonAPICanDeactivateAnUserByID@test.com"
	passwd := "test"
	realm := entities.Realm{
		ID:           uuid.New(),
		Name:         login,
		AllowNewUser: true,
	}
	assert.NoError(t, gormDB.Create(&realm).Error)
	user := entities.User{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
		RealmID:   realm.ID,
		RealmName: realm.Name,
	}
	t.Cleanup(func() {
		cleanup(layout)
		gormDB.Unscoped().Delete(&user, "login = ?", login)
		gormDB.Unscoped().Delete(&realm)
	})

	// create the user
	assert.Nil(t, gormDB.Create(&user).Error)
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		// Name: login,
		UID:   user.ID,
		Realm: realm.Name,
	})
	assert.NoError(t, err)
	req, err := http.NewRequest("DELETE", "/user/deactivate", nil)
	assert.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: "Bearer " + jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	userTrial := &entities.User{}
	assert.NoError(t, gormDB.Unscoped().First(userTrial, user.ID).Error)
	assert.Equal(t, 200, rec.Code)
	body, err := io.ReadAll(rec.Body)
	assert.NoError(t, err)
	trial := http.Cookie{}
	assert.NoError(t, json.Unmarshal(body, &trial))
	strings.Split(trial.Value, "")
	assert.True(t, userTrial.DeletedAt.Valid)
}
