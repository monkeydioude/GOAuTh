package errors

import (
	"GOAuTh/pkg/http/response"
	"net/http"
)

type Err struct {
	code         int
	err          error
	httpCallback func(msg string, w http.ResponseWriter)
}

func (e Err) Error() string {
	return e.err.Error()
}

func BadRequest(err error) Err {
	return Err{
		code:         response.BadRequest,
		err:          err,
		httpCallback: response.Unauthorized,
	}
}

func JWTFormatError(err error) Err {
	return BadRequest(err)
}

func Unauthorized(err error) Err {
	return Err{
		code:         response.UnauthorizedCode,
		err:          err,
		httpCallback: response.Unauthorized,
	}
}

func UnprocessableEntity(err error) Err {
	return Err{
		code:         response.UnprocessableEntityCode,
		err:          err,
		httpCallback: response.UnprocessableEntity,
	}
}

func InternalServerError(err error) Err {
	return Err{
		code:         response.InternalServerErrorCode,
		err:          err,
		httpCallback: response.InternalServerError,
	}
}

func DBError(err error) Err {
	return InternalServerError(err)
}

func (err Err) HTTPResponse(w http.ResponseWriter) {
	err.httpCallback(err.Error(), w)
}

func (err Err) CodeInt32() int32 {
	return int32(err.code)
}

func (err Err) Code() int {
	return err.code
}

func HTTPError(err error, w http.ResponseWriter) {
	errType, ok := err.(Err)
	if !ok {
		response.InternalServerError(err.Error(), w)
		return
	}
	errType.HTTPResponse(w)
}
