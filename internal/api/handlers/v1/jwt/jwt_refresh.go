package jwt

import (
	"log"
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

func Refresh(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	cookie, err := req.Cookie(consts.RefreshTokenCookie)
	if err != nil {
		log.Printf("[%s] ERR while retrieving %s cookie: %s", req.Header.Get(consts.X_REQUEST_ID_LABEL), consts.AuthorizationCookie, err.Error())
		response.Unauthorized("No JWT provided in the request", w)
		return
	}

	at, rt, err := services.JWTRefresh(cookie.Value, *h.AccessTokenFactory, *h.RefreshTokenFactory, h.DB)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}

	http.SetCookie(w, &at)
	http.SetCookie(w, &rt)
	response.Json(rt, w)
}
