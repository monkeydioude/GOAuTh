package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"

	"github.com/stretchr/testify/assert"
)

func TestAPIRoutingSendsSessionRoutesByMethod(t *testing.T) {
	// building the mux panics on conflicting patterns
	app := apiRouting(&handlers.Layout{})

	// no Authorization cookie: the session handlers answer 401
	for _, route := range [][2]string{
		{"GET", "/identity/v1/sessions"},
		{"DELETE", "/identity/v1/sessions"},
		{"DELETE", "/identity/v1/sessions/0b8a3c4e-6f6c-4f4e-9d61-0f3a0e5b1c2d"},
		{"PUT", "/identity/v1/auth/logout"},
	} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(route[0], route[1], nil))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, route)
	}

	// any other method is refused by the pattern
	for _, route := range [][2]string{
		{"POST", "/identity/v1/sessions"},
		{"GET", "/identity/v1/sessions/0b8a3c4e-6f6c-4f4e-9d61-0f3a0e5b1c2d"},
		{"GET", "/identity/v1/auth/logout"},
	} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest(route[0], route[1], nil))
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, route)
	}
}
