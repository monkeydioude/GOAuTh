package services

import (
	stdErr "errors"
	"net/http"
	"strings"
	"time"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"github.com/monkeydioude/goauth/v2/pkg/crypt"
	"github.com/monkeydioude/goauth/v2/pkg/errors"
	"github.com/monkeydioude/goauth/v2/pkg/tools/result"
	"gorm.io/gorm"
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

func JWTStatus(token string, factory JWTFactory) (http.Cookie, error) {
	jwt, err := factory.DecodeToken(token)
	if err != nil {
		return http.Cookie{}, err
	}

	if !JWTClaimsValidation(jwt.Claims) {
		return http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_MISSING_PARAMS))
	}
	if jwt.Claims.Expire < factory.TimeFn().Unix() {
		return http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_EXPIRED))
	}
	return http.Cookie{
		Name:   consts.AuthorizationCookie,
		Value:  "Bearer " + jwt.GetToken(),
		MaxAge: int(jwt.GetExpiresIn().Seconds()),
		Path:   "/",
	}, nil
}

func JWTRefresh(
	token string,
	accessTokenFactory JWTFactory,
	refreshTokenFactory JWTFactory,
	db *gorm.DB,
) (http.Cookie, http.Cookie, error) {
	jwt, err := refreshTokenFactory.DecodeToken(token)
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(err)
	}
	if !JWTClaimsValidation(jwt.Claims) {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKEN_MISSING_PARAMS))
	}
	var user entities.User
	res := db.Select("refresh_token").Where("id = ?", jwt.Claims.UID).First(&user)
	if res.Error != nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(res.Error)
	}
	if user.RefreshToken == nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_MISSING_TOKEN))
	}
	if *user.RefreshToken != jwt.GetToken() {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(stdErr.New(consts.ERR_TOKENS_DONT_MATCH))
	}
	newAT, err := accessTokenFactory.GenerateToken(jwt.Claims)
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(err)
	}
	newRT, err := refreshTokenFactory.TryRefresh(jwt)
	if err != nil {
		return http.Cookie{}, http.Cookie{}, errors.Unauthorized(err)
	}

	return http.Cookie{
			Name:    consts.AuthorizationCookie,
			Value:   "Bearer " + newAT.GetToken(),
			Expires: time.Now().Add(newAT.GetExpiresIn()),
			MaxAge:  int(newAT.GetExpiresIn().Seconds()),
			Path:    "/",
		}, http.Cookie{
			Name:    consts.RefreshTokenCookie,
			Value:   newRT.GetToken(),
			Expires: time.Now().Add(newRT.GetExpiresIn()),
			MaxAge:  int(newRT.GetExpiresIn().Seconds()),
			Path:    "/",
		}, nil
}
