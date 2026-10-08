package crypt

import (
	"regexp"
	"strings"
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

func TestIsAccessKey(t *testing.T) {
	key, err := NewAccessKey()
	assert.NoError(t, err)
	assert.True(t, IsAccessKey(key))
	for _, trial := range []string{"", "gak_", key[:46], key + "a", "gck_" + key[4:], "gak_" + strings.Repeat("!", 43), " " + key} {
		assert.False(t, IsAccessKey(trial), trial)
	}
}

func TestAccessKeyPrefixOfAShortString(t *testing.T) {
	assert.Equal(t, "", AccessKeyPrefixOf("gak_short"))
}
