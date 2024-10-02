package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/errors"
	stdErr "errors"
	"net/http"
	"strings"
)

func GetJWTFromToken(tokenWithBearer string) (string, error) {
	parts := strings.Split(tokenWithBearer, " ")
	partsLen := len(parts)
	if partsLen == 0 || partsLen != 2 || parts[0] != "Bearer" {
		return "", errors.Unauthorized(stdErr.New(consts.ERR_WRONG_TOKEN_SCHEMA))
	}
	return parts[1], nil
}

func JWTStatus(tokenWithBearer string, factory JWTFactory) (http.Cookie, error) {
	token, err := GetJWTFromToken(tokenWithBearer)
	if err != nil {
		return http.Cookie{}, err
	}
	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return http.Cookie{}, err
	}

	if jwt.Claims.RemainingRefresh(factory.TimeFn()) <= 0 {
		return http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_EXPIRED))
	}
	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

func JWTRefresh(tokenWithBearer string, factory JWTFactory) (http.Cookie, error) {
	token, err := GetJWTFromToken(tokenWithBearer)
	if err != nil {
		return http.Cookie{}, err
	}

	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return http.Cookie{}, errors.Unauthorized(err)
	}

	jwt, err = factory.TryRefresh(jwt)
	if err != nil {
		return http.Cookie{}, errors.Unauthorized(err)
	}

	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}
