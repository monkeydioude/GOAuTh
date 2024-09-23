package entities

import (
	"GOAuTh/pkg/crypt"
	"time"
)

type JWT[C crypt.JWTClaims] struct {
	ExpiresIn     time.Duration
	SigningMethod crypt.JWTSigningMethod
	Token         string
	Claims        crypt.JWTDefaultClaims
}

// GetExpiresIn implements entity.JWT.
func (j JWT[C]) GetExpiresIn() time.Duration {
	return j.ExpiresIn
}

// GetSigningMethod implements entity.JWT.
func (j JWT[C]) GetClaims() crypt.JWTDefaultClaims {
	return j.Claims
}

// GetSigningMethod implements entity.JWT.
func (j JWT[C]) GetSigningMethod() crypt.JWTSigningMethod {
	return j.SigningMethod
}

// GetToken implements entity.JWT.
func (j JWT[C]) GetToken() string {
	return j.Token
}
