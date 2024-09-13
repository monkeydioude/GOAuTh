package crypt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
)

// JWTClaims is a mandatory part of the JWT generation.
// Claims make up 2 of the 3 mandatory parts of a JWT
type JWTClaims interface {
	GetClaims() []byte
}

// JWTDefaultClaims should be used as base minimal claims
type JWTDefaultClaims struct {
	Iss  string `json:"iss,omitempty"`
	Exp  int64  `json:"exp"`
	Sub  string `json:"sub,omitempty"`
	Name string `json:"name"`
}

// GetClaims implmentation from the JWTClaims interface
func (c JWTDefaultClaims) GetClaims() []byte {
	claims, err := json.Marshal(c)
	if err != nil {
		log.Printf("[ERR ] getClaims: %s\n", err.Error())
		return []byte("{}")
	}
	return claims
}

// JWTSigningMethod defines which JWT signing method to use
// and how a the method should sign.
// JWT signing methods are a mandatory part of a JWT signing process.
type JWTSigningMethod interface {
	GenerateJWT(data string) []byte
	Name() string
}

type Secret string

// HS256 is the type for the [Hashmac SHA256](https://en.wikipedia.org/wiki/HMAC) JWT signing method.
// It holds the secret used for the signing, and implements JWTSigningMethod
type HS256 Secret

// GenerateJWT implementation of the JWTSigningMethod interface for HS256
func (secret HS256) GenerateJWT(data string) []byte {
	hmac := hmac.New(sha256.New, []byte(secret))
	hmac.Write([]byte(data))
	return hmac.Sum(nil)
}

// Name implementation of the JWTSigningMethod interface for HS256
func (HS256) Name() string {
	return "HS256"
}

// NewJWT generates a new JWT Token using a signing method and claims
func NewJWT(method JWTSigningMethod, p JWTClaims) (string, error) {
	header := fmt.Sprintf(`{"alg":"%s","typ":"JWT"}`, method.Name())
	claims := p.GetClaims()
	hB64 := JWTBase64Encode([]byte(header))
	pB64 := JWTBase64Encode(claims)
	sign := JWTBase64Encode(method.GenerateJWT(fmt.Sprintf("%s.%s", hB64, pB64)))
	return strings.Join([]string{hB64, pB64, sign}, "."), nil
}

// AssertJWT verifies JWT integrity
func AssertJWT(token string, method JWTSigningMethod) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return errors.New("invalid token format")
	}
	sign := JWTBase64Encode(method.GenerateJWT(fmt.Sprintf("%s.%s", parts[0], parts[1])))
	if sign != parts[2] {
		return errors.New("signatures did not match")
	}
	return nil
}

// JWTBase64Encode encodes a []byte into a string
// following the JWT prerequisites:
// - a Base64 encoding following the URLEncoding standard
// - trimimg of any trailing '=' character
func JWTBase64Encode(data []byte) string {
	return strings.TrimRight(base64.
		URLEncoding.
		EncodeToString(data), "=")
}
