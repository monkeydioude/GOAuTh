package crypt

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccessKeyFormat(t *testing.T) {
	key, err := NewAccessKey()
	assert.NoError(t, err)
	assert.Len(t, key, 47)
	// what a secret scanner looks for
	assert.Regexp(t, regexp.MustCompile(`^gak_[A-Za-z0-9_-]{43}$`), key)
	assert.Equal(t, key[4:12], AccessKeyPrefixOf(key))

	other, err := NewAccessKey()
	assert.NoError(t, err)
	assert.NotEqual(t, key, other)
}

func TestAccessKeyPrefixOfAShortString(t *testing.T) {
	assert.Equal(t, "", AccessKeyPrefixOf("gak_short"))
}
