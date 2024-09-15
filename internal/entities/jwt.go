package entities

import (
	"GOAuTh/pkg/crypt"
	"errors"
	"log"
	"time"

	"gorm.io/gorm"
)

type JWT struct {
	ExpiresIn     time.Duration
	SigningMethod crypt.JWTSigningMethod
	Token         string
	Claims        crypt.JWTDefaultClaims
}

// GetSigningMethod implements entities.JWT.
func (j JWT) GetClaims() crypt.JWTClaims {
	return j.Claims
}

// GetSigningMethod implements entities.JWT.
func (j JWT) GetSigningMethod() crypt.JWTSigningMethod {
	return j.SigningMethod
}

// GetTimeReference implements entities.JWT.
func (j JWT) GetTimeReference() time.Time {
	return time.Now()
}

// GetToken implements entities.JWT.
func (j JWT) GetToken() string {
	return j.Token
}

// TryRefresh tries to refresh the token if possible, else returns an error
func (j JWT) TryRefresh(db *gorm.DB) (JWT, error) {
	claims, err := crypt.DecodeJWT[crypt.JWTDefaultClaims](j.Token, j.SigningMethod)
	if err != nil {
		log.Printf("[ERR ] while decoding the JWT: %s", err.Error())
		return JWT{}, errors.New("the JWT is invalid")
	}

	if claims.Refresh >= time.Now().Unix() {
		return JWT{}, errors.New("the JWT is too old to be refreshed")
	}

	if claims.Expire >= time.Now().Unix() {
		// return HandleJWTCreation(claims, signingMethod, jwtParams)
	}
	return j, nil
}

type JWTFactory struct {
	SigningMethod crypt.JWTSigningMethod
	ExpiresIn     time.Duration
	RefreshesIn   time.Duration
	TimeFn        func() time.Time
}

func (jf JWTFactory) GenerateToken(u *DefaultUser) (JWT, error) {
	claims := u.IntoClaims(jf.ExpiresIn, jf.RefreshesIn)
	token, err := crypt.NewJWT(jf.SigningMethod, claims)
	if err != nil {
		return JWT{}, err
	}
	return JWT{
		Claims:        claims,
		Token:         token,
		SigningMethod: jf.SigningMethod,
		ExpiresIn:     jf.ExpiresIn,
	}, nil
}

func (jf JWTFactory) DecodeToken(token string) (JWT, error) {
	claims, err := crypt.DecodeJWT[crypt.JWTDefaultClaims](token, jf.SigningMethod)
	if err != nil {
		return JWT{}, err
	}
	return JWT{
		Claims:        claims,
		Token:         token,
		SigningMethod: jf.SigningMethod,
		ExpiresIn:     jf.ExpiresIn,
	}, nil
}

func NewJWTFactory(
	signingMethod crypt.JWTSigningMethod,
	expiresIn time.Duration,
	refreshesIn time.Duration,
	timeRefFn func() time.Time,
) *JWTFactory {
	return &JWTFactory{
		SigningMethod: signingMethod,
		ExpiresIn:     expiresIn,
		RefreshesIn:   refreshesIn,
		TimeFn:        timeRefFn,
	}
}
