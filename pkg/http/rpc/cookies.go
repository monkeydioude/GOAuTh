package rpc

import (
	"context"
	"errors"

	"fmt"
	"net/http"

	"google.golang.org/grpc/metadata"
)

func SetCookie(cookie http.Cookie) metadata.MD {
	return metadata.Pairs(SetCookieLabel, cookie.String())
}

func AppendCookie(md metadata.MD, cookie http.Cookie) metadata.MD {
	md.Append(SetCookieLabel, cookie.String())
	return md
}

func FetchCookie(headers metadata.MD, key string) (http.Cookie, error) {
	cookies := headers.Get(SetCookieLabel)
	if len(cookies) == 0 {
		return http.Cookie{}, errors.New("no cookies")
	}
	for _, cookieLine := range cookies {
		cookieSlice, err := http.ParseCookie(cookieLine)
		if err != nil {
			continue
		}
		for _, cookie := range cookieSlice {
			if cookie.Name == key {
				return *cookie, nil
			}
		}
	}
	return http.Cookie{}, fmt.Errorf("couldnt find cookie with key: %s", key)
}

func FetchCookieFromContext(ctx context.Context, key string) (http.Cookie, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return http.Cookie{}, errors.New("didnt find any metadata")
	}
	return FetchCookie(md, AuthorizationCookie)
}

func SetOutgoingCookie(ctx context.Context, cookie http.Cookie) context.Context {
	return metadata.NewOutgoingContext(ctx, SetCookie(cookie))
}

func AddOutgoingCookie(ctx context.Context, cookie http.Cookie) context.Context {
	return WriteOutgoingMetas(ctx, [2]string{SetCookieLabel, cookie.String()})
}

func SetIncomingCookie(ctx context.Context, cookie http.Cookie) context.Context {
	return metadata.NewIncomingContext(ctx, SetCookie(cookie))
}
