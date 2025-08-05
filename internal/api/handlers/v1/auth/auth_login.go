package auth

import (
	"log"
	"net/http"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/errors"
	"github.com/monkeydioude/goauth/pkg/http/request"
	"github.com/monkeydioude/goauth/pkg/http/response"
)

// Login would be the route used for logging a user in the a system
func Login(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	rawPayload := request.Json[entities.User](req)
	if rawPayload.IsErr() {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), rawPayload.Error.Error())
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	user := rawPayload.Result()
	if user.RealmName == "" {
		response.BadRequest("realm_name missing", w)
		return
	}
	res, err := services.AuthLogin(user, h.DB, h.UserParams, h.JWTFactory)
	if err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		errors.HTTPError(err, w)
		return
	}
	http.SetCookie(w, &res)
	response.JsonOk(w)
}
