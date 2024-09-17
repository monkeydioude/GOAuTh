package functional_tests

import (
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers/auth"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestICanSignup(t *testing.T) {
	layout, gormDB, _ := setup(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/signup", layout.Post(auth.Signup))

	login := "TestICanSignup@test.com"
	rec := httptest.NewRecorder()
	user := entities.DefaultUser{
		Login:    login,
		Password: "test",
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	body, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("could not marshal: %s", err.Error())
	}
	req, err := http.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("could not query POST /auth/signup: %s", err.Error())
	}
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("response code should have beem 200, was: %d", rec.Code)
	}

	trial := entities.DefaultUser{}
	if err := gormDB.Where("login = ?", login).First(&trial).Error; err != nil {
		t.Fatal(err)
	}
	if trial.Login != login {
		t.Fatalf("should be able to fetch %s login after signup", login)
	}
}
