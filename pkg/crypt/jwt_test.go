package crypt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestICanSignAJWT(t *testing.T) {
	trial := JWTDefaultClaims{
		Expire: time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		UID:    0,
		Realm:  "test1",
		Type:   "test",
	}
	goal := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXBlIjoidGVzdCIsImV4cGlyZSI6MTcyODA4MDU0MiwidWlkIjowLCJyZWFsbSI6InRlc3QxIn0.ZlrudxhikVY8nu_55CC4d4WzOg3i6jIwRJvhVuT_-hA"

	signature, err := NewJWT(HS256("test"), trial)
	assert.NoError(t, err)
	assert.Equal(t, goal, signature)
	res, err := DecodeJWT[JWTDefaultClaims](signature, HS256("test"))
	assert.NoError(t, err)
	assert.Equal(t, trial, res)
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
