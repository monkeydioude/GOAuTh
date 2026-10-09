package user

import (
	"log/slog"
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/monkeydioude/goauth/v2/pkg/http/request"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

func EditLogin(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	cookie, err := req.Cookie(consts.AuthorizationCookie)
	if err != nil {
		slog.WarnContext(req.Context(), "could not retrieve cookie", "cookie", consts.AuthorizationCookie, "error", err.Error())
		response.Unauthorized("No JWT provided in the request", w)
		return
	}
	rawPayload := request.Json[entities.EditUserPayload](req)
	if rawPayload.IsErr() {
		slog.ErrorContext(req.Context(), rawPayload.Error.Error())
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	editUserPayload := rawPayload.Result()
	editUserPayload.UserParams = h.UserParams
	if err := services.UserEditLogin(cookie.Value, h.AccessTokenFactory, h.DB.WithContext(req.Context()), editUserPayload); err != nil {
		slog.ErrorContext(req.Context(), err.Error())
		errors.HTTPError(err, w)
		return
	}
	response.JsonOk(w)
}
