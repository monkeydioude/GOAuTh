package middlewares

import "net/http"

type API struct {
	chain http.Handler
}

func Mux(handler http.Handler) *API {
	return &API{handler}
}

func (a *API) Use(handler func(http.Handler) http.Handler) {
	a.chain = handler(a.chain)
}

func (a *API) UseGroup(handlers ...func(http.Handler) http.Handler) {
	for _, handler := range handlers {
		a.Use(handler)
	}
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.chain.ServeHTTP(w, r)
}
