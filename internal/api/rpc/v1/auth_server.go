package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/internal/domain/models"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}
	grpc.SendHeader(ctx, rpc.AppendCookie(md, res))
	return Ok(), nil
}

func (h *AuthRPCHandler) Deactivate(ctx context.Context, _ *Empty) (*Response, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return BadRequest("didnt find any metadata"), nil
	}
	cookie, err := rpc.FetchCookie(md, consts.AuthorizationCookie)
	if err != nil {
		return BadRequest("could not find token metadata"), nil
	}

	jwt, err := h.JWTFactory.DecodeCookieToken(&cookie)
	if err != nil {
		return InternalServerError("could not decode cookie"), nil
	}
	if jwt.Claims.UID == 0 {
		return BadRequest("no uid in the JWT"), nil

	}
	err = services.AuthDeactivate(jwt.Claims.UID, h.DB)
	if err != nil {
		return InternalServerError("could not deactivate user"), nil
	}
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
