package v1

import (
	"net/http"

	"github.com/monkeydioude/goauth/pkg/errors"
	"github.com/monkeydioude/goauth/pkg/http/response"
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

func Created(message string) *Response {
	return &Response{
		Code:    http.StatusCreated,
		Message: message,
	}
}

func Success(msg string) *Response {
	return &Response{
		Code:    int32(response.OkCode),
		Message: msg,
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

func FromErrToUserActionStatusResponse(err error) *UserActionStatusResponse {
	res := FromErrToResponse(err)
	return &UserActionStatusResponse{
		Code:    res.Code,
		Message: res.Message,
	}
}
