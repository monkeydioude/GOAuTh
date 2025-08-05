package entities

import (
	"time"

	"github.com/monkeydioude/goauth/pkg/crypt"
)

// @todo: implement realm/access system
type Entity interface {
	// GetRealm() string
	// GetLevel() int32
}

type User[C crypt.JWTClaims] interface {
	Entity
	IsRevoked(time.Time) bool
	IntoClaims() C
}
