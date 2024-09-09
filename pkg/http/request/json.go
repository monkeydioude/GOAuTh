package request

import (
	"GOAuTh/pkg/tools/result"
	"encoding/json"
	"io"
	"net/http"
)

func Json[T any](req *http.Request) result.R[T] {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return result.Error[T](err)
	}
	var entity T
	if err := json.Unmarshal(body, &entity); err != nil {
		return result.Error[T](err)
	}
	return result.Ok(&entity)
}
