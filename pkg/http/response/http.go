package response

import (
	"encoding/json"
	"net/http"
)

type HTTPResponse struct {
	Code    int
	Message string
}

const (
	OkCode                  int = 200
	UnauthorizedCode        int = 401
	UnprocessableEntityCode int = 422
	InternalServerErrorCode int = 500
)

func Unauthorized(msg string, w http.ResponseWriter) {
	w.WriteHeader(UnauthorizedCode)
	res, err := json.Marshal(HTTPResponse{
		Code:    UnauthorizedCode,
		Message: msg,
	})
	if err != nil {
		w.Write([]byte("Could not marshal matters"))
		return
	}
	w.Write(res)
	w.Header().Set("Content-Type", "application/json")
}

func UnprocessableEntity(msg string, w http.ResponseWriter) {
	w.WriteHeader(UnprocessableEntityCode)
	res, err := json.Marshal(HTTPResponse{
		Code:    UnprocessableEntityCode,
		Message: msg,
	})
	if err != nil {
		w.Write([]byte("Could not marshal matters"))
		return
	}
	w.Write(res)
	w.Header().Set("Content-Type", "application/json")
}

func InternalServerError(msg string, w http.ResponseWriter) {
	w.WriteHeader(InternalServerErrorCode)
	res, err := json.Marshal(HTTPResponse{
		Code:    InternalServerErrorCode,
		Message: msg,
	})
	if err != nil {
		w.Write([]byte("Could not marshal matters"))
		return
	}
	w.Write(res)
	w.Header().Set("Content-Type", "application/json")
}
