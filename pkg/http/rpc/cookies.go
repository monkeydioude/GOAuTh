package rpc

import (
	"GOAuTh/internal/config/consts"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/grpc/metadata"
)

func SetCookie(cookie http.Cookie) metadata.MD {
	return metadata.Pairs(consts.SetCookie, cookie.String())
}

func AppendCookie(md metadata.MD, cookie http.Cookie) metadata.MD {
	md.Append(consts.SetCookie, cookie.String())
	return md
}

func FetchCookie(headers metadata.MD, key string) (http.Cookie, error) {
	if _, ok := headers[consts.SetCookie]; !ok {
		return http.Cookie{}, errors.New("no cookies")
	}
	for _, cookieLine := range headers[consts.SetCookie] {
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
