package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"

	"google.golang.org/grpc"
)

type JWTRPCHandler struct {
	UnimplementedJWTServer
	JWTFactory *services.JWTFactory
}

func (h *JWTRPCHandler) Status(ctx context.Context, req *JWTRequest) (*Response, error) {
	jwt, err := h.JWTFactory.DecodeToken(req.Token)
	if err != nil {
		return Unauthorized("could not decode token"), err
	}

	res := http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return Ok(), nil
}

func NewJWTRPCHandler(layout *handlers.Layout) *JWTRPCHandler {
	return &JWTRPCHandler{
		JWTFactory: layout.JWTFactory,
	}
}
