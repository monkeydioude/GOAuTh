package functional_tests

import (
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers/jwt"
	"GOAuTh/pkg/crypt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestICanRefreshAValidToken(t *testing.T) {
	layout, gormDB, _ := setup(t)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second

	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))

	login := "TestICanRefreshAValidToken@test.com"
	passwd := "test"
	user := entities.DefaultUser{
		Login:     login,
		Password:  passwd,
		RevokedAt: nil,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	err := layout.HydrateEntity(&user)
	if err != nil {
		t.Fatal(err)
	}
	if err := gormDB.Save(&user).Error; err != nil {
		t.Fatalf("could not save user: %s", err.Error())
	}
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	if err != nil {
		t.Fatalf("could not GenerateToken: %s", err.Error())
	}

	timeRef := layout.JWTFactory.TimeFn()
	// we go 5s forward in time, so we are in refresh spot
	// (expiresIn 3s, refreshesIn 10s, 5 is between, so it's cool)
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef.Add(5 * time.Second)
	}
	req, err := http.NewRequest("POST", "/jwt/refresh", nil)
	if err != nil {
		t.Fatalf("could not query POST /jwt/refresh: %s", err.Error())
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("response code should have been 200, was: %d", rec.Code)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("could not read buffer: %s", err.Error())
	}
	trial := http.Cookie{}
	if err := json.Unmarshal(body, &trial); err != nil {
		t.Fatalf("could not unmarshal body: %s", err.Error())
	}

	jwt2, err := layout.JWTFactory.DecodeToken(trial.Value)
	if err != nil {
		t.Fatalf("could not decode refresheed token: %s", err.Error())
	}
	// 5 + 3 seconds, because we went 5s forward in time, and JWTFactory ExpiresIn config is 3s
	if jwt2.Token == jwt.Token || jwt2.Claims.Expire != timeRef.Add((5+3)*time.Second).Unix() {
		t.Fatal("status returned the same token or a different expire date")
	}
}

func TestIGetA401OnRefreshingAnInvalidToken(t *testing.T) {
	layout, _, _ := setup(t)
	// enforce ExpiresIn and RefreshesIn in a clear and wanted context
	layout.JWTFactory.ExpiresIn = 3 * time.Second
	layout.JWTFactory.RefreshesIn = 10 * time.Second
	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/refresh", layout.Post(jwt.Refresh))

	login := "TestIGetA401OnRefreshingAnInvalidToken@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	if err != nil {
		t.Fatalf("could not GenerateToken: %s", err.Error())
	}

	timeRef := layout.JWTFactory.TimeFn()
	// we go 12s forward in time, so we are beyond refresh time.
	// RefreshesIn 10s when we generated thee token
	layout.JWTFactory.TimeFn = func() time.Time {
		return timeRef.Add(12 * time.Second)
	}

	req, err := http.NewRequest("POST", "/jwt/refresh", nil)
	if err != nil {
		t.Fatalf("could not query POST /jwt/refresh: %s", err.Error())
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	// should fail
	if rec.Code != 401 {
		t.Fatalf("response code should have been 401, was: %d", rec.Code)
	}
}
