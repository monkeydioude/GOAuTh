package entity

import (
	"GOAuTh/pkg/crypt"
	"time"
)

// JWT defines a standard JSON Web Token.
//
// JWT takes a single generic, so we can provide and use custom Claims
// while letting the compiler asserting the type for us.
type JWT[C crypt.JWTClaims] interface {
	GetSigningMethod() crypt.JWTSigningMethod
	GetToken() string
	GetClaims() C
	GetExpiresIn() time.Duration
}

// JWTFactory defines a standard JSON Web Token factory.
//
// JWTFactory takes a single generic, so we can provide and use custom Claims
// while letting the compiler asserting the type for us.
type JWTFactory[C crypt.JWTClaims, J JWT[C]] interface {
	GenerateToken(C) (J, error)
	DecodeToken(string) (J, error)
}
