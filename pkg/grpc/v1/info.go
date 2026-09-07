package v1

import (
	"context"
	"fmt"

	"github.com/calqs/gopkg/dt"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"gorm.io/gorm"
)

type InfoHandler struct {
	UnimplementedInfoServer
	DB *gorm.DB
}

func (h *InfoHandler) GetUsers(ctx context.Context, req *UserFilters) (*UsersInfo, error) {
	users, err := services.GetUsersInfo(&services.GetsUsersInfoIn{
		Login: req.Login,
		Realm: req.Realm,
	}, h.DB)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %v", err)
	}
	users_info := dt.SliceTransform(users, func(user entities.User) *UserInfo {
		if user.Realm == nil {
			return nil
		}
		return &UserInfo{
			Login: user.Login,
			Realm: user.Realm.Name,
		}
	})
	return &UsersInfo{Users: users_info}, nil
}
