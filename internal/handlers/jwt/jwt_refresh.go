package jwt

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/http/response"
	"log"
	"net/http"
	"time"
)

func Refresh(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
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

	jwt, err = h.JWTFactory.TryRefresh(jwt)
	if err != nil {
		response.Unauthorized(err.Error(), w)
		return
	}
	res := &http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.Claims.RemainingRefresh(time.Now()).Seconds()),
		Path:   "/",
	}
	http.SetCookie(w, res)
	response.Json(res, w)
}
