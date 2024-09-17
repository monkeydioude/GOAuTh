package functional_tests

import (
	"GOAuTh/internal/handlers/jwt"
	"GOAuTh/pkg/crypt"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestICanGetAValidTokensStatus(t *testing.T) {
	layout, _, _ := setup(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/status", layout.Post(jwt.Status))

	login := "TestICanGetAValidTokensStatus@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	if err != nil {
		t.Fatalf("could not GenerateToken: %s", err.Error())
	}

	req, err := http.NewRequest("POST", "/jwt/status", nil)
	if err != nil {
		t.Fatalf("could not query POST /jwt/status: %s", err.Error())
	}
	req.AddCookie(&http.Cookie{
		Name:  "Authorization",
		Value: jwt.Token,
	})
	mux.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("response code should have been 200, was: %d, %s", rec.Code, rec.Body)
	}

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("could not read buffer: %s", err.Error())
	}
	trial := http.Cookie{}
	if err := json.Unmarshal(body, &trial); err != nil {
		t.Fatalf("could not unmarshal body: %s", err.Error())
	}
	if trial.Value != jwt.Token {
		t.Fatal("status returned a diffeerent token")
	}
}

func TestIGetA401OnInvalidToken(t *testing.T) {
	layout, _, _ := setup(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/jwt/status", layout.Post(jwt.Status))

	login := "TestIGetA401OnInvalidToken@test.com"
	rec := httptest.NewRecorder()

	jwt, err := layout.JWTFactory.GenerateToken(crypt.JWTDefaultClaims{
		Name: login,
	})
	if err != nil {
		t.Fatalf("could not GenerateToken: %s", err.Error())
	}
	// assign an invalid value
	jwt.Token = "bs-value"

	req, err := http.NewRequest("POST", "/jwt/status", nil)
	if err != nil {
		t.Fatalf("could not query POST /jwt/status: %s", err.Error())
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

	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("could not read buffer: %s", err.Error())
	}
	trial := http.Cookie{}
	if err := json.Unmarshal(body, &trial); err != nil {
		t.Fatalf("could not unmarshal body: %s", err.Error())
	}
	if trial.Value == jwt.Token {
		t.Fatal("status returned a diffeerent token")
	}
}
