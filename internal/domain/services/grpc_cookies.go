package services

import (
	"context"
	"net/http"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/pkg/http/rpc"
)

func SetAuthorizationTokenMetaOut(ctx context.Context, token string) context.Context {
	return rpc.SetOutgoingCookie(ctx, http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: token,
	})
}

func SetAuthorizationTokenMetaIn(ctx context.Context, token string) context.Context {
	return rpc.SetIncomingCookie(ctx, http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: token,
	})
}
