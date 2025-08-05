package services

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/monkeydioude/goauth/internal/config/consts"
	"github.com/monkeydioude/goauth/internal/domain/entities"
	"github.com/monkeydioude/goauth/pkg/crypt"
)

const (
	INVALID_JWT_ERR           = "invalid JWT"
	TOO_OLD_JWT_ERR           = "the JWT is too old to be refreshed"
	REVOKED_OR_REVOCATION_ERR = "token is revoked, or an error during revocation check happened: %s"
)

// entities.JWTFactory produces, decodes and tries to refresh entities.JWTs.
//
// ExpiresIn should always be smaller than RefreshesIn
// since an expired token can still be refreshed if
// in the RefreshesIn threshold.
//
// TimeFn is a function made for providing the moment in time
// for date comparison. It may be as simple as:
//
// ...
//
//	TimeFn: func () time.Time {
//	    return time.Now()
//	}
//
// ...
type JWTFactory struct {
	SigningMethod crypt.JWTSigningMethod

	// ExpiresIn < RefreshesIn
	ExpiresIn time.Duration

	// RefreshesIn > ExpiresIn
	RefreshesIn time.Duration

	TimeFn              func() time.Time
	RevocationCheckerFn func(uint, func() time.Time) (bool, error)
}

// GenerateToken implements "github.com/monkeydioude/goauth/pkg/domain/entities".JWTFactory
func (jf JWTFactory) GenerateToken(claims crypt.JWTDefaultClaims) (entities.JWT[crypt.JWTDefaultClaims], error) {
	timeRef := jf.TimeFn()
	claims.Expire = timeRef.Add(jf.ExpiresIn).Unix()
	claims.Refresh = timeRef.Add(jf.RefreshesIn).Unix()

	token, err := crypt.NewJWT(jf.SigningMethod, claims)
	if err != nil {
		return entities.JWT[crypt.JWTDefaultClaims]{}, err
	}
	return entities.JWT[crypt.JWTDefaultClaims]{
		Claims:        claims,
		Token:         token,
		SigningMethod: jf.SigningMethod,
		ExpiresIn:     jf.ExpiresIn,
	}, nil
}

// DecodeToken implements "github.com/monkeydioude/goauth/pkg/domain/entities".JWTFactory.
func (jf JWTFactory) DecodeToken(token string) (entities.JWT[crypt.JWTDefaultClaims], error) {
	claims, err := crypt.DecodeJWT[crypt.JWTDefaultClaims](token, jf.SigningMethod)
	if err != nil {
		return entities.JWT[crypt.JWTDefaultClaims]{}, err
	}
	return entities.JWT[crypt.JWTDefaultClaims]{
		Claims:        claims,
		Token:         token,
		SigningMethod: jf.SigningMethod,
		ExpiresIn:     jf.ExpiresIn,
	}, nil
}

func (jf JWTFactory) DecodeCookieToken(tokCook *http.Cookie) (entities.JWT[crypt.JWTDefaultClaims], error) {
	parts := strings.Split(tokCook.Value, " ")
	partsLen := len(parts)
	if partsLen != 2 || (partsLen > 0 && parts[0] != "Bearer") {
		return entities.JWT[crypt.JWTDefaultClaims]{}, errors.New(consts.ERR_WRONG_TOKEN_SCHEMA)
	}
	return jf.DecodeToken(parts[1])
}

// TryRefresh tries to refresh the token if possible, else returns an error
func (jf JWTFactory) TryRefresh(j entities.JWT[crypt.JWTDefaultClaims]) (entities.JWT[crypt.JWTDefaultClaims], error) {
	claims, err := crypt.DecodeJWT[crypt.JWTDefaultClaims](j.Token, j.SigningMethod)
	if err != nil {
		log.Printf("[ERR ] while decoding a entities.JWT: %s", err.Error())
		return entities.JWT[crypt.JWTDefaultClaims]{}, errors.New(INVALID_JWT_ERR)
	}

	timeRef := jf.TimeFn()
	// too old to be refreshed => exit with error
	if timeRef.After(time.Unix(claims.Refresh, 0)) {
		return entities.JWT[crypt.JWTDefaultClaims]{}, errors.New(TOO_OLD_JWT_ERR)
	}
	// expired but not too old to be refreshed
	if timeRef.After(time.Unix(claims.Expire, 0)) {
		revoked, err := jf.RevocationCheckerFn(j.Claims.UID, jf.TimeFn)
		if err != nil || revoked {
			return entities.JWT[crypt.JWTDefaultClaims]{}, fmt.Errorf(REVOKED_OR_REVOCATION_ERR, err)
		}
		return jf.GenerateToken(claims)
	}
	return j, nil
}

func NewEmptyJWTFactory() *JWTFactory {
	return &JWTFactory{
		TimeFn: func() time.Time {
			return time.Now()
		},
		RevocationCheckerFn: func(_ uint, _ func() time.Time) (bool, error) {
			return true, nil
		},
	}
}

func NewJWTFactory(
	signingMethod crypt.JWTSigningMethod,
	expiresIn time.Duration,
	refreshesIn time.Duration,
	timeRefFn func() time.Time,
	revocationCheckerFn func(uint, func() time.Time) (bool, error),
) *JWTFactory {
	return &JWTFactory{
		SigningMethod:       signingMethod,
		ExpiresIn:           expiresIn,
		RefreshesIn:         refreshesIn,
		TimeFn:              timeRefFn,
		RevocationCheckerFn: revocationCheckerFn,
	}
}
