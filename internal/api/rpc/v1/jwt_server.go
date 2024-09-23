package v1

import (
	"GOAuTh/internal/api/handlers"
	"GOAuTh/internal/domain/services"
	"GOAuTh/pkg/http/rpc"
	"context"

	"google.golang.org/grpc"
)

type JWTRPCHandler struct {
	UnimplementedJWTServer
	JWTFactory *services.JWTFactory
}

func (h *JWTRPCHandler) Status(ctx context.Context, req *JWTRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), nil
	}
	res, err := services.JWTStatus(req.Token, *h.JWTFactory)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return Ok(), nil
}

func (h *JWTRPCHandler) Refresh(ctx context.Context, req *JWTRequest) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), nil
	}
	res, err := services.JWTRefresh(req.Token, *h.JWTFactory)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return Ok(), nil
}

func NewJWTRPCHandler(layout *handlers.Layout) *JWTRPCHandler {
	return &JWTRPCHandler{
		JWTFactory: layout.JWTFactory,
	}
}
