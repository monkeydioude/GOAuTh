package response

import (
	"encoding/json"
	"net/http"
)

func Json[T any](data T, w http.ResponseWriter) {
	res, err := json.Marshal(data)
	if err != nil {
		InternalServerError("Could not marshal json matters", w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(OkCode)
	w.Write(res)
}

func JsonOk(w http.ResponseWriter) {
	JsonMsg(w, "Ok")
}

func JsonMsg(w http.ResponseWriter, msg string) {
	Json(HTTPResponse{
		Code:    OkCode,
		Message: msg,
	}, w)
}
