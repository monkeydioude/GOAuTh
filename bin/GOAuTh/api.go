package main

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/api/handlers/v1/auth"
	"GOAuTh/internal/api/handlers/v1/jwt"
	"GOAuTh/internal/api/handlers/v1/user"
	"GOAuTh/internal/config/boot"
	"GOAuTh/internal/config/middleware"
	"GOAuTh/pkg/http/middlewares"
	"net/http"
	"time"
)

func healthcheck(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"health\": \"OK\"}"))
}

func apiRouting(layout *handlers.Layout) http.Handler {
	mux := http.NewServeMux()
	// routes definition
	// Auth
	mux.HandleFunc("/identity/v1/auth/signup", layout.Post(auth.Signup))
	mux.HandleFunc("/identity/v1/auth/login", layout.Put(auth.Login))
	// User
	mux.HandleFunc("/identity/v1/user/password", layout.Put(user.EditPassword))
	mux.HandleFunc("/identity/v1/user/login", layout.Put(user.EditLogin))
	mux.HandleFunc("/identity/v1/user/deactivate", layout.Delete(user.Deactivate))
	// JWT
	mux.HandleFunc("/identity/v1/jwt/status", layout.Get(jwt.Status))
	mux.HandleFunc("/identity/v1/jwt/refresh", layout.Put(jwt.Refresh))
	// Healthcheck
	mux.HandleFunc("/identity/healthcheck", healthcheck)

	app := middlewares.Mux(mux)
	app.Use(
		middleware.APILogRequest,
		middleware.APIXRequestID,
	)
	return app
}

func setupAPIServer(settings *boot.Settings) *http.Server {
	// setup multiplexer
	mux := apiRouting(settings.Layout)
	return &http.Server{
		Addr:              settings.Api.Port,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           mux,
	}
}
