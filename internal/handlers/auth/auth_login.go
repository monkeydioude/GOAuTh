package auth

import (
	"GOAuTh/internal/consts"
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/http/request"
	"GOAuTh/pkg/http/response"
	"net/http"
	"time"
)

// LogIn would be the route used for logging a user in the a system
func LogIn(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	rawPayload := request.Json[entities.DefaultUser](req)
	if rawPayload.IsErr() {
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}

	payload := rawPayload.Result()
	if payload.IsRevoked(time.Now()) {
		response.Unauthorized("user's access was revoked", w)
		return
	}
	h.HydrateEntity(payload)
	if err := payload.AssertAuth(h.DB); err != nil {
		response.Unauthorized("unauthorized for this login and password", w)
		return
	}
	sign, err := h.JWTFactory.GenerateToken(payload.IntoClaims())
	if err != nil {
		response.InternalServerError("error during jwt generation", w)
		return
	}
	res := &http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  sign.GetToken(),
		MaxAge: int(sign.GetExpiresIn().Seconds()),
		Path:   "/",
	}
	http.SetCookie(w, res)
	response.Json(res, w)
}
