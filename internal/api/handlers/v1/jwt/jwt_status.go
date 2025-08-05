package jwt

import (
	"log"
	"net/http"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/http/response"
)

func Status(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		log.Printf("[%s] ERR while retrieving %s cookie: %s", req.Header.Get(consts.X_REQUEST_ID_LABEL), consts.AuthorizationCookie, err.Error())
		response.Unauthorized("No JWT provided in the request", w)
		return
	}

	res, err := services.JWTStatus(cookie.Value, *h.JWTFactory)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}
	http.SetCookie(w, &res)
	response.JsonOk(w)
}
