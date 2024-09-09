package handlers

import (
	"GOAuTh/pkg/constraints"
	"fmt"
	"net/http"

	"gorm.io/gorm"
)

type Layout struct {
	DB              *gorm.DB
	LoginConstraint constraints.EntityField
}

type Handler func(*Layout, http.ResponseWriter, *http.Request)

var Methods = [5]string{"GET", "POST", "PUT", "PATCH", "DELETE"}

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

func (l *Layout) Get(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("GET", handler)
}

func (l *Layout) Post(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("POST", handler)
}

func (l *Layout) Put(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PUT", handler)
}

func (l *Layout) Patch(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("PATCH", handler)
}

func (l *Layout) Delete(handler Handler) func(http.ResponseWriter, *http.Request) {
	return l.WithMethod("DELETE", handler)
}
