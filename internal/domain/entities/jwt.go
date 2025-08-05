package entities

import (
	"time"

	"github.com/monkeydioude/goauth/pkg/crypt"
)

type JWT[C crypt.JWTClaims] struct {
	ExpiresIn     time.Duration
	SigningMethod crypt.JWTSigningMethod
	Token         string
	Claims        C
}

// GetExpiresIn implements entity.JWT.
func (j JWT[C]) GetExpiresIn() time.Duration {
	return j.ExpiresIn
}

// GetSigningMethod implements entity.JWT.
func (j JWT[C]) GetClaims() C {
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
