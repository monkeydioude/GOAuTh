package auth

import (
	"GOAuTh/internal/handlers"
	"GOAuTh/pkg/entities"
	"GOAuTh/pkg/http/request"
	"GOAuTh/pkg/http/response"
	"net/http"
)

func Signup(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	rawPayload := request.Json[entities.DefaultUser](req)
	if rawPayload.IsErr() {
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	payload := rawPayload.Result()

	if err := h.LoginConstraint(payload.Login); err != nil {
		response.UnprocessableContent(err.Error(), w)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "Authorization",
		Value:  "jwt",
		MaxAge: 60,
		Path:   "/",
	})
	response.JsonOk(w)
}
