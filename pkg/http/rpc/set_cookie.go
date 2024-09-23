package rpc

import (
	"net/http"

	"google.golang.org/grpc/metadata"
)

func SetCookie(cookie http.Cookie) metadata.MD {
	return metadata.Pairs("set-cookie", cookie.String())
}

func AppendCookie(md metadata.MD, cookie http.Cookie) metadata.MD {
	md.Append("set-cookie", cookie.String())
	return md
}
