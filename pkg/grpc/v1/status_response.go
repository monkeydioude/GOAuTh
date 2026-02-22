package v1

import (
	"github.com/monkeydioude/goauth/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func StatusBadRequest(msg string) error {
	return status.Error(codes.InvalidArgument, msg)
}

func StatusUnauthorized(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

func StatusInternalServerError(msg string) error {
	return status.Error(codes.Internal, msg)
}

func StatusNotFound(msg string) error {
	return status.Error(codes.NotFound, msg)
}

func StatusFromErr(err error) error {
	errType, ok := err.(errors.Err)
	if !ok {
		return StatusInternalServerError(err.Error())
	}
	return status.Error(codes.Code(errType.CodeInt32()), errType.Error())
}
