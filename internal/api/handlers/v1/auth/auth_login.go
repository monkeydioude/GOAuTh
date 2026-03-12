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

type LoginIn struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	RealmName string `json:"realm_name"`
}

// Login would be the route used for logging a user in the a system
func Login(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	if h == nil || req == nil {
		response.InternalServerError("no layout or req pointer", w)
		return
	}
	rawPayload := request.Json[LoginIn](req)
	if rawPayload.IsErr() {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), rawPayload.Error.Error())
		response.InternalServerError(rawPayload.Error.Error(), w)
		return
	}
	dto := rawPayload.Result()
	if dto.RealmName == "" {
		response.BadRequest("realm_name missing", w)
		return
	}
	user := entities.User{
		Login:     dto.Login,
		Password:  dto.Password,
		RealmName: dto.RealmName,
	}
	accessToken, refreshToken, err := services.AuthLogin(&user, h.DB, h.UserParams, h.AccessTokenFactory, h.RefreshTokenFactory)
	if err != nil {
		log.Printf("[%s] ERR %s\n", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		errors.HTTPError(err, w)
		return
	}
	http.SetCookie(w, &accessToken)
	http.SetCookie(w, &refreshToken)
	response.JsonOk(w)
}
