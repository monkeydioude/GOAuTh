package middleware

import (
	"log"
	"net/http"
)

func APILogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("incoming API call on %s", r.URL)
		handler.ServeHTTP(w, r)
	})
}
