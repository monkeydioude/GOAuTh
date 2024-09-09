package result

import (
	"errors"
	"log"
)

type R[T any] struct {
	result    *T
	Error     error
	unwrapped bool
}

func Error[T any](err error) R[T] {
	return R[T]{
		Error:     err,
		unwrapped: false,
	}
}

func ErrorString[T any](errStr string) R[T] {
	return R[T]{
		Error:     errors.New(errStr),
		unwrapped: false,
	}
}

func Ok[T any](result *T) R[T] {
	if result == nil {
		return ErrorString[T]("nil result")
	}
	return R[T]{
		result:    result,
		Error:     nil,
		unwrapped: false,
	}
}

func (r *R[T]) IsOk() bool {
	r.unwrapped = true
	return r.Error == nil
}

func (r *R[T]) IsErr() bool {
	return !r.IsOk()
}

func (r R[T]) Result() *T {
	if !r.unwrapped {
		log.Println("[WARN] Result struct: assert Result can be used by calling IsOk() or IsErr() first")
	}
	if r.IsErr() {
		log.Println("[WARN] Result struct: Result contains an error")
	}
	return r.result
}
