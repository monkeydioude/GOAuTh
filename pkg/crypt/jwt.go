package crypt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	go_errors "errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/monkeydioude/goauth/pkg/errors"
)

// JWTClaims is a mandatory part of the JWT generation.
// Claims make up 2 of the 3 mandatory parts of a JWT
type JWTClaims interface {
	GetRawClaims() []byte
}

// JWTDefaultClaims should be used as base minimal claims
type JWTDefaultClaims struct {
	Expire  int64  `json:"expire"`
	Refresh int64  `json:"refresh,omitempty"`
	UID     uint   `json:"uid"`
	Realm   string `json:"realm"`
}

// RemainingRefresh returns the remaining available refresh time.Duration.
// timeRef can be time.Now()
func (c JWTDefaultClaims) RemainingRefresh(timeRef time.Time) time.Duration {
	return time.Unix(c.Refresh, 0).Sub(timeRef)
}

// GetClaims implmentation from the JWTClaims interface
func (c JWTDefaultClaims) GetRawClaims() []byte {
	claims, err := json.Marshal(c)
	if err != nil {
		log.Printf("[ERR ] getClaims: %s\n", err.Error())
		return []byte("{}")
	}
	return claims
}

func (c JWTDefaultClaims) String() string {
	return string(c.GetRawClaims())
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
	claims := p.GetRawClaims()
	hB64 := JWTBase64Encode([]byte(header))
	pB64 := JWTBase64Encode(claims)
	sign := JWTBase64Encode(method.GenerateJWT(fmt.Sprintf("%s.%s", hB64, pB64)))
	return strings.Join([]string{hB64, pB64, sign}, "."), nil
}

// DecodeJWT verifies JWT integrity
func DecodeJWT[T JWTClaims](token string, method JWTSigningMethod) (T, error) {
	parts := strings.Split(token, ".")
	var claims T
	if len(parts) != 3 {
		return claims, errors.JWTFormatError(go_errors.New("invalid token format"))
	}
	sign := JWTBase64Encode(method.GenerateJWT(fmt.Sprintf("%s.%s", parts[0], parts[1])))
	if sign != parts[2] {
		return claims, errors.JWTFormatError(go_errors.New("signatures did not match"))
	}
	claimsb64, err := Decodechunk(parts[1])
	if err != nil {
		return claims, err
	}

	err = json.Unmarshal(claimsb64, &claims)
	if err != nil {
		return claims, err
	}

	return claims, nil
}

func Decodechunk(chunk string) ([]byte, error) {
	// pad only if number of chars is not a multiple of 4
	if l := len(chunk) % 4; l > 0 {
		chunk += strings.Repeat("=", 4-l)
	}
	return base64.URLEncoding.DecodeString(chunk)
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
