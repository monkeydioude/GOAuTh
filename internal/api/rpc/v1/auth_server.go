package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/models"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"GOAuTh/pkg/plugins"
	"context"
	"encoding/json"
	"errors"

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
	user := entities.NewUser(req.Login, req.Password)
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
	user := entities.NewUser(req.Login, req.Password)
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

func NewAuthRPCHandler(layout *handlers.Layout) *AuthRPCHandler {
	return &AuthRPCHandler{
		UserParams: layout.UserParams,
		DB:         layout.DB,
		JWTFactory: layout.JWTFactory,
		Plugins:    layout.Plugins,
	}
}
