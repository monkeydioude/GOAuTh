package crypt

import (
	"crypto/rand"
	"encoding/base64"
	"regexp"
)

const (
	// AccessKeyPrefix starts every access key, so a leaked one is easy to spot.
	AccessKeyPrefix = "gak_"
	// AccessKeyDisplayLength is how many characters after the prefix tell keys
	// apart to a person, without giving a key away.
	AccessKeyDisplayLength = 8
	accessKeyBytes         = 32
)

// NewAccessKey is a fresh access key: the prefix and 256 random bits, base64url
// without padding, 47 characters in all. Only its hash (HashToken) gets kept.
func NewAccessKey() (string, error) {
	raw := make([]byte, accessKeyBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return AccessKeyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

var accessKeyFormat = regexp.MustCompile(`^gak_[A-Za-z0-9_-]{43}$`)

// IsAccessKey tells whether s has the shape of a key NewAccessKey makes, which
// spares a lookup for anything that cannot be one.
func IsAccessKey(s string) bool {
	return accessKeyFormat.MatchString(s)
}

// AccessKeyPrefixOf is the part of a key kept in clear to tell keys apart: its
// first characters after AccessKeyPrefix.
func AccessKeyPrefixOf(key string) string {
	start, end := len(AccessKeyPrefix), len(AccessKeyPrefix)+AccessKeyDisplayLength
	if len(key) < end {
		return ""
	}
	return key[start:end]
}
