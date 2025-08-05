package services

import (
	"context"
	"net/http"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/pkg/http/rpc"
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
