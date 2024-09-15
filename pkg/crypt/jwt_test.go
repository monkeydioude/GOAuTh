package crypt

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestICanSignAJWT(t *testing.T) {
	trials := []JWTDefaultClaims{{
		Expire: time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		Name:   "test@test.com",
	}, {
		Expire:  time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		Refresh: time.Date(2024, 10, 04, 23, 22, 22, 22, time.UTC).Unix(),
		Name:    "test@test.com",
	}}
	goals := []string{
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.0RbVgcJ7ZuMjfXwvbZjkrKG-5HQ2-NgSGKHUWn3_oeM",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsInJlZnJlc2giOjE3MjgwODQxNDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.3BbtoytKfHJUgBacvCMznJEksTJlluzPegzbaLzi8Vk",
	}

	for i, goal := range goals {
		signature, err := NewJWT(HS256("test"), trials[i])
		if err != nil || signature != goal {
			fmt.Println(err, signature != goal)
			t.Fail()
		}

	}
}

func TestICanAssertATokenWasNotTemperedWith(t *testing.T) {
	trial := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.0RbVgcJ7ZuMjfXwvbZjkrKG-5HQ2-NgSGKHUWn3_oeM"
	_, err := DecodeJWT[JWTDefaultClaims](trial, HS256("test"))
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
}

func TestIFailOnTemperedToken(t *testing.T) {
	trial := "ohno.eyJleHAiOjE3MjgwODA1NDIwMDAsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.zcxR5WqM-pxTVWc36Jsl0hwVHyGhaFiHy54BVLkVX9U"
	_, err := DecodeJWT[JWTDefaultClaims](trial, HS256("test"))
	if err == nil {
		t.Fail()
	}
}

func TestICanDecodeAJWT(t *testing.T) {
	goal := JWTDefaultClaims{
		Expire: time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).Unix(),
		Name:   "test@test.com",
	}
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHBpcmUiOjE3MjgwODA1NDIsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.0RbVgcJ7ZuMjfXwvbZjkrKG-5HQ2-NgSGKHUWn3_oeM"
	trial, err := DecodeJWT[JWTDefaultClaims](token, HS256("test"))
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	if !reflect.DeepEqual(goal, trial) {
		t.Fail()
	}
}
