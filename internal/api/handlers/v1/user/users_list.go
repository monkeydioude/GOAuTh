package user

import (
	"log"
	"net/http"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/http/response"
)

type UserInfoOut struct {
	Login string `json:"login"`
	Realm string `json:"realm"`
}

type UsersInfoOut struct {
	Users []*UserInfoOut `json:"users"`
}

func List(h *handlers.Layout, w http.ResponseWriter, req *http.Request) {
	login := req.FormValue("login")
	realm := req.FormValue("realm")
	users, err := services.GetUsersInfo(&services.GetsUsersInfoIn{
		Login: &login,
		Realm: &realm,
	}, h.DB)
	if err != nil {
		log.Printf("[%s] ERR while retrieving users list: %s", req.Header.Get(consts.X_REQUEST_ID_LABEL), err.Error())
		response.InternalServerError("No JWT provided in the request", w)
		return
	}
	usersInfos := dt.SliceTransform(users, func(user entities.User) *UserInfoOut {
		if user.Realm == nil {
			return nil
		}
		return &UserInfoOut{
			Login: user.Login,
			Realm: user.Realm.Name,
		}
	})
	response.Json(UsersInfoOut{Users: usersInfos}, w)
}
