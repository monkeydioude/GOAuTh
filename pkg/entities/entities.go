package entities

import (
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

type Parameters interface {
	// GetPasswordSalt() []byte
	// GetArgon2Params() crypt.Argon2Bag
}

type User[T Parameters] interface {
	Entity[T]
	GetLogin() string
	GetPassword() string
	GetRevokedAt() *time.Time
}
