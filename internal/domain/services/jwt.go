package services

import (
	"GOAuTh/internal/config/consts"
	"GOAuTh/internal/domain/entities"
	"GOAuTh/pkg/crypt"
	"GOAuTh/pkg/errors"
	"GOAuTh/pkg/tools/result"
	stdErr "errors"
	"net/http"
	"strings"
)

func GetTokenFromBearer(tokenWithBearer string) (string, error) {
	parts := strings.Split(tokenWithBearer, " ")
	partsLen := len(parts)
	if partsLen == 0 || partsLen != 2 || parts[0] != "Bearer" {
		return "", errors.Unauthorized(stdErr.New(consts.ERR_WRONG_TOKEN_SCHEMA))
	}
	return parts[1], nil
}

func GetJWTFromBearer(tokenWithBearer string, factory *JWTFactory) result.R[entities.JWT[crypt.JWTDefaultClaims]] {
	token, err := GetTokenFromBearer(tokenWithBearer)
	if err != nil {
		return result.Error[entities.JWT[crypt.JWTDefaultClaims]](err)
	}
	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return result.Error[entities.JWT[crypt.JWTDefaultClaims]](errors.Unauthorized(err))
	}
	return result.Ok(&jwt)
}

func JWTStatus(tokenWithBearer string, factory JWTFactory) (http.Cookie, error) {
	token, err := GetTokenFromBearer(tokenWithBearer)
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
		Value:  "Bearer " + jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

func JWTRefresh(tokenWithBearer string, factory JWTFactory) (http.Cookie, error) {
	token, err := GetTokenFromBearer(tokenWithBearer)
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
		Value:  "Bearer " + jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}
