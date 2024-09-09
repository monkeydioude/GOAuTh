package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"GOAuTh/internal/boot"
	"GOAuTh/internal/handlers"
	"GOAuTh/internal/handlers/auth"
	"GOAuTh/internal/handlers/jwt"
	"GOAuTh/pkg/constraints"
	"GOAuTh/pkg/entities"
)

func routing(handlers *handlers.Layout) *http.ServeMux {
	mux := http.NewServeMux()

	// routes definition
	mux.HandleFunc("/auth/signup", handlers.Post(auth.Signup))
	mux.HandleFunc("/auth/login", handlers.Put(auth.LogIn))
	mux.HandleFunc("/jwt/status", handlers.Get(jwt.Status))
	return mux
}

func setupServer(api boot.Api, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", api.Port),
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           mux,
	}
}

func main() {
	res := boot.Please(
		[]any{&entities.DefaultUser{}},
		constraints.EmailConstraint,
	)
	if res.IsErr() {
		log.Fatal(res.Error)
	}

	settings := res.Result()
	// setup multiplexer
	mux := routing(settings.Layout)
	// server definition
	server := setupServer(settings.Api, mux)

	// Start the server on port
	log.Println("Server starting on port", settings.Api.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
