package entity

import (
	"GOAuTh/pkg/crypt"
	"time"
)

type User[C crypt.JWTClaims] interface {
	IsRevoked(time.Time) bool
	IntoClaims() C
}
