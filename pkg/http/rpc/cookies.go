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

func SetCookies(cookies ...http.Cookie) metadata.MD {
	md := metadata.Pairs()
	for _, cookie := range cookies {
		md = AppendCookie(md, cookie)
	}
	return md
}

func AppendCookies(md metadata.MD, cookies ...http.Cookie) metadata.MD {
	for _, cookie := range cookies {
		md = AppendCookie(md, cookie)
	}
	return md
}

func FetchCookie(headers metadata.MD, key string) (http.Cookie, error) {
	cookies := headers.Get(SetCookieLabel)
	if len(cookies) == 0 {
		return http.Cookie{}, errors.New("no cookies")
	}
	for _, cookieLine := range cookies {
		header := http.Header{}
		header.Add("Set-Cookie", cookieLine)
		resp := http.Response{Header: header}
		for _, cookie := range resp.Cookies() {
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
	return FetchCookie(md, key)
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
