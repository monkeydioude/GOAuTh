package user

import (
	"log"
	"net/http"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/errors"
	"github.com/monkeydioude/goauth/pkg/http/response"
)

// Login would be the route used for logging a user in the a system
func Deactivate(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
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

	jwt, err := h.JWTFactory.DecodeCookieToken(cookie)
	if err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		response.Unauthorized(err.Error(), w)
		return
	}
	if !services.JWTClaimsValidation(jwt.Claims) {
		response.Unauthorized(consts.ERR_INVALID_CREDENTIALS, w)
		return
	}
	err = services.AuthDeactivate(jwt.Claims.UID, h.DB)
	if err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		errors.HTTPError(err, w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: "",
	})
	response.JsonOk(w)
}
