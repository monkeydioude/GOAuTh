package response

import (
	"encoding/json"
	"net/http"
)

type HTTPResponse struct {
	Code    int
	Message string
}

func UnprocessableContent(msg string, w http.ResponseWriter) {
	w.WriteHeader(422)
	res, err := json.Marshal(HTTPResponse{
		Code:    422,
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
	w.WriteHeader(500)
	res, err := json.Marshal(HTTPResponse{
		Code:    500,
		Message: msg,
	})
	if err != nil {
		w.Write([]byte("Could not marshal matters"))
		return
	}
	w.Write(res)
	w.Header().Set("Content-Type", "application/json")
}
