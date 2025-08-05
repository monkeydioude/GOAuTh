package auth

import (
	"net/http"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/errors"
	"github.com/monkeydioude/goauth/pkg/http/request"
	"github.com/monkeydioude/goauth/pkg/http/response"
	"github.com/monkeydioude/goauth/pkg/plugins"
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
	if user.RealmName == "" {
		response.BadRequest("realm_name missing", w)
		return
	}

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
