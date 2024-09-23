package jwt

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/response"
	"log"
	"net/http"
)

func Status(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		log.Printf("[ERR ] while retrieving %s cookie: %s", consts.AuthorizationCookie, err.Error())
		response.Unauthorized("No JWT provided in the request", w)
		return
	}

	jwt, err := h.JWTFactory.DecodeToken(cookie.Value)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}

	res := &http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}
	http.SetCookie(w, res)
	response.Json(res, w)
}