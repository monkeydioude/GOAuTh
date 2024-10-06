package auth

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

	res, err := services.AuthLogin(rawPayload.Result(), h.DB, h.UserParams, h.JWTFactory)
	if err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		errors.HTTPError(err, w)
		return
	}
	http.SetCookie(w, &res)
	response.Json(res, w)
}
