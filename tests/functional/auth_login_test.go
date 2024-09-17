package functional_tests

import (
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers/auth"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestICanLogin(t *testing.T) {
	layout, gormDB, timeRef := setup(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", layout.Post(auth.LogIn))
	rec := httptest.NewRecorder()

	// define the user
	login := "TestICanLogin@test.com"
	passwd := "test"
	user := entities.DefaultUser{
		Login:    login,
		Password: passwd,
	}
	defer gormDB.Unscoped().Delete(&user, "login = ?", login)

	// create the user
	err := layout.HydrateEntity(&user)
	if err != nil {
		t.Fatal(err)
	}
	if err := gormDB.Save(&user).Error; err != nil {
		t.Fatal(err.Error())
	}
	// restore password to its previous state, before it got modified by gorm
	user.Password = passwd

	// prepare to request the endpoint
	body, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("could not marshal: %s", err.Error())
	}
	req, err := http.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("could not query POST /auth/login: %s", err.Error())
	}
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("response code should have been 200, was: %d", rec.Code)
	}

	// retrieve the token from the response
	cookies, err := http.ParseCookie(rec.Result().Header["Set-Cookie"][0])
	if err != nil {
		t.Fatalf("could not ParseCookie: %s", err)
	}
	responseToken := cookies[0].Value
	trialJWT, err := layout.JWTFactory.DecodeToken(responseToken)
	if err != nil {
		t.Fatalf("could not DecodeToken: %s", err.Error())
	}
	if trialJWT.Claims.Name != login ||
		trialJWT.Claims.Expire != int64(timeRef.Add(3*time.Second).Unix()) ||
		trialJWT.Claims.Refresh != int64(timeRef.Add(10*time.Second).Unix()) {
		t.Fatal("token did not match")
	}
}
