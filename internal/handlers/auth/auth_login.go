package auth

import (
	"GOAuTh/internal/entities"
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/crypt"
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
	h.HydrateEntity(payload)
	if err := payload.AssertAuth(h.DB); err != nil {
		response.Unauthorized("unauthorized for this login and password", w)
		return
	}
	sign, err := crypt.NewJWT(h.SigningMethod, crypt.JWTDefaultClaims{Name: payload.Login, Exp: time.Now().Add(3 * time.Minute).UnixMilli()})
	if err != nil {
		response.InternalServerError("error during jwt generation", w)
		return
	}
	res := &http.Cookie{
		Name:   "Authorization",
		Value:  sign,
		MaxAge: 60,
		Path:   "/",
	}
	http.SetCookie(w, res)
	response.Json(res, w)
}
