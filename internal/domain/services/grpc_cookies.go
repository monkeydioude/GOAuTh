package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"
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
