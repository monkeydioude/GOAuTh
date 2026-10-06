package services

import (
	"testing"
	"time"

	"github.com/monkeydioude/goauth/v2/pkg/crypt"

	"github.com/stretchr/testify/assert"
)

func TestFactoryCanGenerateAndDecodeAToken(t *testing.T) {
	jf := NewJWTFactory(crypt.HS256("test"), 1*time.Second, func() time.Time {
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}, func(uint, func() time.Time) (bool, error) {
		return false, nil
	}, "test")

	jwt, err := jf.GenerateToken(crypt.JWTDefaultClaims{})
	if err != nil {
		t.Fail()
	}

	jwt2, err := jf.DecodeToken(jwt.Token)
	if err != nil || jwt2.Claims.Expire != time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC).Add(time.Second).Unix() {
		t.Fail()
	}
}

func TestFactoryCanRefreshAToken(t *testing.T) {
	timeRefFn := func() time.Time {
		// 2024-10-04 22:22:22
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}
	revocCheckerFn := func(uint, func() time.Time) (bool, error) {
		return false, nil
	}
	// expire time is 2024-10-04 22:22:27
	// expire in 5s
	jf := NewJWTFactory(crypt.HS256("test"), 5*time.Second, timeRefFn, revocCheckerFn, "test")

	jwt1, err := jf.GenerateToken(crypt.JWTDefaultClaims{})
	if err != nil {
		t.Fail()
	}
	// since TimeFn is the time reference for generating tokens,
	// would be time.Now() most of the time, we move the factory's
	// time ref forward in time, to pretend time advanced.
	// Before 2024-10-04 22:22:22, now 2024-10-04 22:22:32
	jf.TimeFn = func() time.Time {
		// 2024-10-04 22:22:32
		return timeRefFn().Add(10 * time.Second)
	}
	trial, err := jf.TryRefresh(jwt1)
	// fail if err
	if err != nil ||
		// fail if same token
		trial.Token == jwt1.Token ||
		// trial.Claims.Expire should be equal to time.Date(2024, 10, 04, 22, 22, 32, 0, time.UTC) + 5 * time.Second
		// since we use jf1 to refresh jwt2
		trial.Claims.Expire != time.Date(2024, 10, 04, 22, 22, 37, 0, time.UTC).Unix() {
		t.Fail()
	}
}

func TestFactoryGeneratesUniqueTokensFromTheSameClaims(t *testing.T) {
	// a frozen clock: both tokens are generated in the same second
	jf := NewJWTFactory(crypt.HS256("test"), 1*time.Second, func() time.Time {
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}, func(uint, func() time.Time) (bool, error) {
		return false, nil
	}, "test")
	claims := crypt.JWTDefaultClaims{UID: 1, Realm: "test", SID: "session-1"}

	jwt1, err := jf.GenerateToken(claims)
	assert.NoError(t, err)
	jwt2, err := jf.GenerateToken(claims)
	assert.NoError(t, err)

	assert.NotEqual(t, jwt1.Token, jwt2.Token)
	assert.NotEmpty(t, jwt1.Claims.JTI)
	assert.NotEqual(t, jwt1.Claims.JTI, jwt2.Claims.JTI)
}

func TestFactoryRefreshKeepsTheSessionAndChangesTheJTI(t *testing.T) {
	jf := NewJWTFactory(crypt.HS256("test"), 5*time.Second, func() time.Time {
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}, func(uint, func() time.Time) (bool, error) {
		return false, nil
	}, "test")
	jwt1, err := jf.GenerateToken(crypt.JWTDefaultClaims{UID: 1, Realm: "test", SID: "session-1"})
	assert.NoError(t, err)

	trial, err := jf.TryRefresh(jwt1)
	assert.NoError(t, err)

	assert.Equal(t, "session-1", trial.Claims.SID)
	assert.NotEmpty(t, trial.Claims.JTI)
	assert.NotEqual(t, jwt1.Claims.JTI, trial.Claims.JTI)
}
