package auth

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/errors"
	"GOAuTh/pkg/http/request"
	"GOAuTh/pkg/http/response"
	"GOAuTh/pkg/plugins"
	"net/http"
)

// Signup would be the route user when creating a new user
func Signup(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	rawPayload := request.Json[entities.User](req)
	if rawPayload.IsErr() {
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	user := rawPayload.Result()
	h.Plugins.TriggerBefore(plugins.OnUserCreation, nil)
	err := services.AuthSignup(user, h.UserParams, h.DB)
	if err != nil {
		errors.HTTPError(err, w)
		return
	}
	user.Password = ""
	h.Plugins.TriggerAfter(plugins.OnUserCreation, user)
	response.Json(user, w)
}
