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
	w.WriteHeader(200)
	w.Write(res)
}

func JsonOk(w http.ResponseWriter) {
	Json(HTTPResponse{
		Code:    200,
		Message: "Ok",
	}, w)
}
