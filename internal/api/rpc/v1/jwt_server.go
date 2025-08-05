package v1

import (
	"context"

	"github.com/monkeydioude/goauth/internal/api/handlers"
	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/services"
	"github.com/monkeydioude/goauth/pkg/http/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type JWTRPCHandler struct {
	UnimplementedJWTServer
	JWTFactory *services.JWTFactory
}

func (h *JWTRPCHandler) Status(ctx context.Context, req *Empty) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return BadRequest("didnt find any metadata"), nil
	}
	cookie, err := rpc.FetchCookie(md, consts.AuthorizationCookie)
	if err != nil {
		return BadRequest("could not find token metadata"), nil
	}
	res, err := services.JWTStatus(cookie.Value, *h.JWTFactory)
	if err != nil {
		return FromErrToResponse(err), nil
	}
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return Ok(), nil
}

func (h *JWTRPCHandler) Refresh(ctx context.Context, req *Empty) (*Response, error) {
	if req == nil {
		return InternalServerError("no req pointer"), nil
	}
	cookie, err := rpc.FetchCookieFromContext(ctx, consts.AuthorizationCookie)
	if err != nil {
		return BadRequest(err.Error()), nil
	}
	res, err := services.JWTRefresh(cookie.Value, *h.JWTFactory)
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
