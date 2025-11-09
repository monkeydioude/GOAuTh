package v1

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/internal/domain/models"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/http/rpc"
	"github.com/monkeydioude/goauth/pkg/plugins"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

type AuthRPCHandler struct {
	UnimplementedAuthServer
	UserParams *models.UsersParams
	DB         *gorm.DB
	JWTFactory *services.JWTFactory
	Plugins    *plugins.PluginsRecord
}

func (h *AuthRPCHandler) Signup(ctx context.Context, req *UserRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), errors.New("no req pointer")
	}
	h.Plugins.TriggerBefore(plugins.OnUserCreation, nil)
	user := entities.NewUser(req.Login, req.Password, req.Realm)
	err := services.AuthSignup(user, h.UserParams, h.DB)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	user.Password = ""
	resp, err := json.Marshal(user)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	h.Plugins.TriggerAfter(plugins.OnUserCreation, user)
	return Success(string(resp)), nil
}

func (h *AuthRPCHandler) Login(ctx context.Context, req *UserRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), errors.New("no req pointer")
	}
	user := entities.NewUser(req.Login, req.Password, req.Realm)
	res, err := services.AuthLogin(user, h.DB, h.UserParams, h.JWTFactory)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	grpc.SendHeader(ctx, rpc.AppendCookie(md, res))
	return Ok(), nil
}

func (h *AuthRPCHandler) Delete(ctx context.Context, req *AuthIdRequest) (*Response, error) {
	return Ok(), h.DB.Delete(&entities.User{}, "id = ?", req.Uid).Error
}

func NewAuthRPCHandler(layout *handlers.Layout) *AuthRPCHandler {
	return &AuthRPCHandler{
		UserParams: layout.UserParams,
		DB:         layout.DB,
		JWTFactory: layout.JWTFactory,
		Plugins:    layout.Plugins,
	}
}
