package handlers

import (
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/internal/domain/models"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/crypt"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

// Layout is the context (in a setting sort of way) of a handler.
// It mostly holds dependencies and settings that need to get passed on
// up the execution tree.
type Layout struct {
	DB              *gorm.DB
	LoginConstraint constraints.EntityField
	SigningMethod   crypt.JWTSigningMethod
	UserParams      *models.UsersParams
	JWTFactory      *services.JWTFactory
}

// Handler our basic generic route handler
type Handler func(*Layout, http.ResponseWriter, *http.Request)

// Methods vector of available HTTP MEthods
var Methods = [5]string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// WithMethod is a geeneric wrapper around a generic handler, forcing the a HTTP verb
func (l *Layout) WithMethod(method string, handler Handler) func(http.ResponseWriter, *http.Request) {
	// #StephenCurrying
	return func(w http.ResponseWriter, req *http.Request) {
		for _, m := range Methods {
			// a method matches
			if m == method {
				handler(l, w, req)
				return
			}
		}
		// no method matched the one provided over the array of available methods
		w.WriteHeader(405)
		w.Write([]byte(fmt.Sprintf("Method %s not allowd", req.Method)))
	}
}

// Delete is a wrapper around a generic handler, forcing the GET HTTP verb
func (l *Layout) Get(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("GET", handler)
}

// Delete is a wrapper around a generic handler, forcing the POST HTTP verb
func (l *Layout) Post(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("POST", handler)
}

// Delete is a wrapper around a generic handler, forcing the PUT HTTP verb
func (l *Layout) Put(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PUT", handler)
}

// Delete is a wrapper around a generic handler, forcing the PATCH HTTP verb
func (l *Layout) Patch(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PATCH", handler)
}

// Delete is a wrapper around a generic handler, forcing the DELETE HTTP verb
func (l *Layout) Delete(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("DELETE", handler)
}
