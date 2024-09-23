package v1

import "GOAuTh/pkg/http/response"

func Unauthorized(msg string) *Response {
	return &Response{
		Code:    int32(response.UnauthorizedCode),
		Message: msg,
	}
}

func Ok() *Response {
	return &Response{
		Code:    int32(response.OkCode),
		Message: "Ok",
	}
}
