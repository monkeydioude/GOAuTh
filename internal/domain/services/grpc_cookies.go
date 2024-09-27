package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/http/rpc"
	"context"
	"net/http"
)

func AddAuthorizationTokenMetadata(ctx context.Context, token string) context.Context {
	return rpc.WriteOutgoingCookie(ctx, http.Cookie{
		Name:  consts.AuthorizationCookie,
		Value: token,
	})
}
