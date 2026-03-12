package v1

import (
	"context"

	"github.com/monkeydioude/goauth/v2/internal/api/handlers"
	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/services"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/timed"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"
	"gorm.io/gorm"

	"google.golang.org/grpc"
)

type JWTRPCHandler struct {
	UnimplementedJWTServer
	AccessTokenFactory  *services.JWTFactory
	RefreshTokenFactory *services.JWTFactory
	DB                  *gorm.DB
}

func (h *JWTRPCHandler) Status(ctx context.Context, req *StatusIn) (*StatusOut, error) {
	if req == nil {
		return nil, StatusInternalServerError("no req pointer")
	}
	token, err := FetchAccessToken(ctx, req)
	if err != nil {
		return nil, StatusBadRequest("could not find access token in metadata or payload")
	}
	res, err := services.JWTStatus(token, *h.AccessTokenFactory)
	if err != nil {
		return nil, StatusFromErr(err)
	}
	grpc.SendHeader(ctx, rpc.SetCookie(res))
	return &StatusOut{
		AccessTokenValid: res.Value != "",
		// RefreshTokenValid: true,
	}, nil
}

func (h *JWTRPCHandler) Refresh(ctx context.Context, req *RefreshIn) (*RefreshOut, error) {
	if req == nil {
		return nil, StatusInternalServerError("no req pointer")
	}
	token := req.RefreshToken
	cookie, err := rpc.FetchCookieFromContext(ctx, consts.RefreshTokenCookie)
	if err != nil {
		if token == "" {
			return nil, StatusBadRequest(err.Error())
		}
	} else {
		token = cookie.Value
	}
	atf := h.AccessTokenFactory
	if req.AccessExpiresInSeconds != nil {
		atf = atf.WithExpiresIn(timed.Seconds(*req.AccessExpiresInSeconds))
	}
	rtf := h.RefreshTokenFactory
	if req.RefreshExpiresInSeconds != nil {
		rtf = rtf.WithExpiresIn(timed.Seconds(*req.RefreshExpiresInSeconds))
	}
	accessToken, refreshToken, err := services.JWTRefresh(token, *atf, *rtf, h.DB)
	if err != nil {
		return nil, StatusFromErr(err)
	}
	grpc.SendHeader(ctx, rpc.SetCookies(accessToken, refreshToken))
	return &RefreshOut{
		AccessToken:      accessToken.Value,
		AccessExpiresAt:  accessToken.Expires.Unix(),
		RefreshToken:     refreshToken.Value,
		RefreshExpiresAt: refreshToken.Expires.Unix(),
	}, nil
}

func NewJWTRPCHandler(layout *handlers.Layout) *JWTRPCHandler {
	return &JWTRPCHandler{
		AccessTokenFactory:  layout.AccessTokenFactory,
		RefreshTokenFactory: layout.RefreshTokenFactory,
		DB:                  layout.DB,
	}
}
