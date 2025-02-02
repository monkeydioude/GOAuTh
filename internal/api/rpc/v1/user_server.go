package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/models"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"context"

	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

type UserRPCHandler struct {
	UnimplementedUserServer
	UserParams *models.UsersParams
	DB         *gorm.DB
	JWTFactory *services.JWTFactory
}

func (h *UserRPCHandler) Deactivate(ctx context.Context, _ *Empty) (*Response, error) {
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
	if !services.JWTClaimsValidation(jwt.Claims) {
		return Unauthorized(consts.ERR_INVALID_CREDENTIALS), nil
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

func NewUserRPCHandler(layout *handlers.Layout) *UserRPCHandler {
	return &UserRPCHandler{
		UserParams: layout.UserParams,
		DB:         layout.DB,
		JWTFactory: layout.JWTFactory,
	}
}
