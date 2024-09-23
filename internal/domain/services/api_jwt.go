package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/pkg/errors"
	"net/http"
)

func JWTStatus(token string, factory JWTFactory) (http.Cookie, error) {
	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return http.Cookie{}, err
	}

	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

func JWTRefresh(token string, factory JWTFactory) (http.Cookie, error) {
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
