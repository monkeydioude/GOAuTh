package crypt

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns the hex-encoded SHA-256 of a token, which is what gets stored instead of the token itself.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
