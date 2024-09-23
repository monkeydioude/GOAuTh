package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/api/handlers/v1/auth"
	"GOAuTh/internal/api/handlers/v1/jwt"
	v1 "GOAuTh/internal/api/rpc/v1"
	"GOAuTh/internal/config/boot"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/entities/constraints"

	"google.golang.org/grpc"
)

func routing(layout *handlers.Layout) *http.ServeMux {
	mux := http.NewServeMux()

	// routes definition
	mux.HandleFunc("/v1/auth/signup", layout.Post(auth.Signup))
	mux.HandleFunc("/v1/auth/login", layout.Put(auth.Login))
	mux.HandleFunc("/v1/jwt/status", layout.Get(jwt.Status))
	return mux
}

func setupServer(api *boot.Api, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%s", api.Port),
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		Handler:           mux,
	}
}

func setupGRPC(layout *handlers.Layout) *grpc.Server {
	server := grpc.NewServer()
	v1.RegisterJWTServer(server, v1.NewJWTRPCHandler(layout))
	return server
}

func main() {
	res := boot.Please(
		[]any{entities.NewEmptyUser()},
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
