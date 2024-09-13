package crypt

import (
	"fmt"
	"testing"
	"time"
)

func TestICanSignAJWT(t *testing.T) {
	p := JWTDefaultClaims{
		Exp:  time.Date(2024, 10, 04, 22, 22, 22, 22, time.UTC).UnixMilli(),
		Name: "test@test.com",
	}
	goal := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjgwODA1NDIwMDAsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.zcxR5WqM-pxTVWc36Jsl0hwVHyGhaFiHy54BVLkVX9U"
	signature, err := NewJWT(HS256("test"), p)
	if err != nil || signature != goal {
		t.FailNow()
	}
}

func TestICanAssertATokenWasNotTemperedWith(t *testing.T) {
	trial := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MjgwODA1NDIwMDAsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.zcxR5WqM-pxTVWc36Jsl0hwVHyGhaFiHy54BVLkVX9U"
	if err := AssertJWT(trial, HS256("test")); err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestIFailOnTemperedToken(t *testing.T) {
	trial := "ohno.eyJleHAiOjE3MjgwODA1NDIwMDAsIm5hbWUiOiJ0ZXN0QHRlc3QuY29tIn0.zcxR5WqM-pxTVWc36Jsl0hwVHyGhaFiHy54BVLkVX9U"
	if err := AssertJWT(trial, HS256("test")); err == nil {
		t.FailNow()
	}
}
