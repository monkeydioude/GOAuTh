package v1

import (
	"GOAuTh/pkg/errors"
	"GOAuTh/pkg/http/response"
)

func BadRequest(msg string) *Response {
	return &Response{
		Code:    int32(response.BadRequestCode),
		Message: msg,
	}
}
func Unauthorized(msg string) *Response {
	return &Response{
		Code:    int32(response.UnauthorizedCode),
		Message: msg,
	}
}
func InternalServerError(msg string) *Response {
	return &Response{
		Code:    int32(response.InternalServerErrorCode),
		Message: msg,
	}
}

func Ok() *Response {
	return &Response{
		Code:    int32(response.OkCode),
		Message: "Ok",
	}
}

func FromErrToResponse(err error) *Response {
	errType, ok := err.(errors.Err)
	if !ok {
		return InternalServerError(err.Error())
	}
	return &Response{
		Code:    errType.CodeInt32(),
		Message: errType.Error(),
	}
}
