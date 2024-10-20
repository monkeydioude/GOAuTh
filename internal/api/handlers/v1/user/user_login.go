package user

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/errors"
	"GOAuTh/pkg/http/request"
	"GOAuTh/pkg/http/response"
	"log"
	"net/http"
)

func EditLogin(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
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
	rawPayload := request.Json[entities.EditUserPayload](req)
	if rawPayload.IsErr() {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), rawPayload.Error.Error())
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	editUserPayload := rawPayload.Result()
	editUserPayload.UserParams = h.UserParams
	if err := services.UserEditLogin(cookie.Value, h.JWTFactory, h.DB, editUserPayload); err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		errors.HTTPError(err, w)
		return
	}
	response.JsonOk(w)
}
