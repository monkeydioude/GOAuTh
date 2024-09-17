package entities

import (
	"GOAuTh/pkg/crypt"
	"time"
)

// Entity defines a general entity
type Entity[T Parameters] interface {
	// GetCreatedAt() time.Time
	// GetUpdatedAt() time.Time
	// GetDeletedAt() result.R[*time.Time]
	// Some config parameters, strongly related to the kind of entity using
	// e.g.: a A type of entity won't get the same parameters as a B type
	SetParameters(Parameters) error
}

// Parameters is no better than "any" for now. Might remove.
type Parameters interface {
}

type User[C crypt.JWTClaims] interface {
	IsRevoked(time.Time) bool
	IntoClaims() C
}

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
