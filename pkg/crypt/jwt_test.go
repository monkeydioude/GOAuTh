package crypt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICanSignAJWT(t *testing.T) {
	trials := []JWTDefaultClaims{{
		Expire: time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		UID:    0,
	}, {
		Expire:  time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		Refresh: time.Date(2024, 10, 04, 23, 22, 22, 22, time.UTC).Unix(),
		UID:     0,
	}}
	goals := []string{
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsInVpZCI6MH0.3JcNwhA08nShDFyw2ekyrYOM8g8svv6sOX2bUbUpbWU",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsInJlZnJlc2giOjE3MjgwODQxNDIsInVpZCI6MH0.pIWEavbtqDwl_Nz7LPyQqgGtMSWC-82d9MJqiq7WM-Q",
	}

	for i, goal := range goals {
		signature, err := NewJWT(HS256("test"), trials[i])
		assert.NoError(t, err)
		assert.Equal(t, goal, signature)
	}
}

func TestICanAssertATokenWasNotTemperedWith(t *testing.T) {
	trial := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.0RbVgcJ7ZuMjfXwvbZjkrKG-5HQ2-NgSGKHUWn3_oeM"
	_, err := DecodeJWT[JWTDefaultClaims](trial, HS256("test"))
	assert.NoError(t, err)
}

func TestIFailOnTemperedToken(t *testing.T) {
	trial := "ohno.eyJleHAiOjE3MjgwODA1NDIwMDAsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.zcxR5WqM-pxTVWc36Jsl0hwVHyGhaFiHy54BVLkVX9U"
	_, err := DecodeJWT[JWTDefaultClaims](trial, HS256("test"))
	assert.Error(t, err)
}

func TestICanDecodeAJWT(t *testing.T) {
	goal := JWTDefaultClaims{
		Expire: time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
	}
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.0RbVgcJ7ZuMjfXwvbZjkrKG-5HQ2-NgSGKHUWn3_oeM"
	trial, err := DecodeJWT[JWTDefaultClaims](token, HS256("test"))
	assert.NoError(t, err)
	assert.Equal(t, goal, trial)
}
