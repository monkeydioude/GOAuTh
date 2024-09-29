package middlewares

import "net/http"

type API struct {
	chain http.Handler
}

func Mux(handler http.Handler) *API {
	return &API{handler}
}

func (a *API) Use(handlers ...func(http.Handler) http.Handler) {
	for _, handler := range handlers {
		a.chain = handler(a.chain)
	}
}
func (a *API) UseBefore(handlers ...func(http.ResponseWriter, *http.Request)) {
	for _, handler := range handlers {
		a.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler(w, r)
				h.ServeHTTP(w, r)
			})
		})
	}
}
func (a *API) UseAfter(handlers ...func(http.ResponseWriter, *http.Request)) {
	for _, handler := range handlers {
		a.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				h.ServeHTTP(w, r)
				handler(w, r)
			})
		})
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.chain.ServeHTTP(w, r)
}
