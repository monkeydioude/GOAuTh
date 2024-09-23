package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/internal/domain/models"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"context"
	"errors"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

type AuthRPCHandler struct {
	UnimplementedAuthServer
	LoginConstraint constraints.EntityField
	UserParams      *models.UsersParams
	DB              *gorm.DB
	JWTFactory      *services.JWTFactory
}

func (h *AuthRPCHandler) Signup(ctx context.Context, req *UserRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), errors.New("no req pointer")
	}
	user := entities.NewUser(req.Login, req.Password)
	err := services.AuthSignup(user, h.LoginConstraint, h.DB)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	return Ok(), nil
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
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return Ok(), nil
}

func NewAuthRPCHandler(layout *handlers.Layout) *AuthRPCHandler {
	return &AuthRPCHandler{
		LoginConstraint: layout.LoginConstraint,
		UserParams:      layout.UserParams,
		DB:              layout.DB,
		JWTFactory:      layout.JWTFactory,
	}
}
