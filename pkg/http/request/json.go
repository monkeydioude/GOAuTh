package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/monkeydioude/goauth/pkg/tools/result"
)

func Json[T any](req *http.Request) result.R[T] {
	if req == nil {
		return result.Error[T](errors.New("nil *http.Request"))
	}
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
