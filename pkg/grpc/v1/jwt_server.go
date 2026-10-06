package v1

import (
	"context"
	"net/http"
	"time"

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
	SessionReuseGrace   time.Duration
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
	// refresh_expires_in_seconds is ignored: a refresh token lives as long as its session
	res, err := services.JWTRefresh(token, req.GetClient().IntoClientInfo(), *atf, *h.RefreshTokenFactory, h.SessionReuseGrace, h.DB)
	if err != nil {
		return nil, StatusFromErr(err)
	}
	out := &RefreshOut{
		AccessToken:      res.AccessToken.Value,
		AccessExpiresAt:  res.AccessToken.Expires.Unix(),
		RefreshExpiresAt: res.SessionExpires.Unix(),
	}
	cookies := []http.Cookie{res.AccessToken}
	// nil when a parallel refresh already rotated the token: the caller keeps the one it got
	if res.RefreshToken != nil {
		out.RefreshToken = res.RefreshToken.Value
		cookies = append(cookies, *res.RefreshToken)
	}
	grpc.SendHeader(ctx, rpc.SetCookies(cookies...))
	return out, nil
}

func NewJWTRPCHandler(layout *handlers.Layout) *JWTRPCHandler {
	return &JWTRPCHandler{
		AccessTokenFactory:  layout.AccessTokenFactory,
		RefreshTokenFactory: layout.RefreshTokenFactory,
		DB:                  layout.DB,
		SessionReuseGrace:   layout.SessionReuseGrace,
	}
}
