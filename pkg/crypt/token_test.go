package crypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTokenIsAStableHexSHA256(t *testing.T) {
	// SHA-256 test vector from FIPS 180-2
	assert.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", HashToken("abc"))
	assert.Len(t, HashToken("eyJhbGciOiJIUzI1NiJ9.e30.signature"), 64)
	assert.Equal(t, HashToken("same token"), HashToken("same token"))
	assert.NotEqual(t, HashToken("token a"), HashToken("token b"))
}
