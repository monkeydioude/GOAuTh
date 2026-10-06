package v1

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/internal/domain/models"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"
	"github.com/monkeydioude/goauth/v2/pkg/plugins"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type AuthRPCHandler struct {
	UnimplementedAuthServer
	UserParams          *models.UsersParams
	DB                  *gorm.DB
	AccessTokenFactory  *services.JWTFactory
	RefreshTokenFactory *services.JWTFactory
	Plugins             *plugins.PluginsRecord
	MaxActiveSessions   int
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
	atf := h.AccessTokenFactory
	if req.AccessExpiresInSeconds != nil {
		atf = h.AccessTokenFactory.WithExpiresIn(time.Second * time.Duration(*req.AccessExpiresInSeconds))
	}
	// refresh_expires_in_seconds is ignored: a refresh token lives as long as its session
	accessToken, refreshToken, err := services.AuthLogin(user, req.GetClient().IntoClientInfo(), h.DB, h.UserParams, atf, h.RefreshTokenFactory, h.MaxActiveSessions)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	md := rpc.SetCookies(accessToken, refreshToken)
	grpc.SendHeader(ctx, md)
	return Ok(), nil
}

func (h *AuthRPCHandler) Delete(ctx context.Context, req *AuthIdRequest) (*Response, error) {
	return Ok(), h.DB.Delete(&entities.User{}, "id = ?", req.Uid).Error
}

func NewAuthRPCHandler(layout *handlers.Layout) *AuthRPCHandler {
	return &AuthRPCHandler{
		UserParams:          layout.UserParams,
		DB:                  layout.DB,
		AccessTokenFactory:  layout.AccessTokenFactory,
		RefreshTokenFactory: layout.RefreshTokenFactory,
		Plugins:             layout.Plugins,
		MaxActiveSessions:   layout.MaxActiveSessions,
	}
}

func (h *AuthRPCHandler) Logout(ctx context.Context, req *LogoutRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), errors.New("no req pointer")
	}
	return Ok(), h.DB.Model(&entities.User{}).Where("id = ? AND realm_id = (?)", req.Uid, h.DB.Table("realms").Select("id").Where("name = ?", req.Realm)).Update("refresh_token", nil).Error
}
