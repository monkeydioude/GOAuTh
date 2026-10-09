package jwt

import (
	"log/slog"
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

func Status(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		slog.WarnContext(req.Context(), "could not retrieve cookie", "cookie", consts.AuthorizationCookie, "error", err.Error())
		response.Unauthorized("No JWT provided in the request", w)
		return
	}

	tok, err := services.GetTokenFromBearer(cookie.Value)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}
	res, err := services.JWTStatus(req.Context(), tok, *h.AccessTokenFactory)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}
	http.SetCookie(w, &res)
	response.JsonOk(w)
}
