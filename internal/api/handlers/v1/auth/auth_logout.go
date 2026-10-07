package auth

import (
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

// Logout ends the caller's session and clears both cookies. The session is named
// by the Refresh cookie, else by the access token, which may have expired.
func Logout(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	token, factory, err := logoutToken(h, req)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	jwt, err := services.SessionOfToken(token, *factory)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	// a session already revoked or expired is left as is
	if err := services.LogoutSession(h.DB, jwt.Claims, factory.TimeFn()); err != nil {
		errors.HTTPError(errors.DBError(err), w)
		return
	}
	for _, name := range []string{consts.AuthorizationCookie, consts.RefreshTokenCookie} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
	}
	response.JsonOk(w)
}

// logoutToken picks the token naming the caller's session, with the factory reading it.
func logoutToken(h *handlers.Layout, req *http.Request) (string, *services.JWTFactory, error) {
	if cookie, err := req.Cookie(consts.RefreshTokenCookie); err == nil && cookie.Value != "" {
		return cookie.Value, h.RefreshTokenFactory, nil
	}
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		return "", nil, errors.Unauthorized(err)
	}
	token, err := services.GetTokenFromBearer(cookie.Value)
	return token, h.AccessTokenFactory, err
}
