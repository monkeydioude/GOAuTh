package entities

import (
	"GOAuTh/pkg/crypt"
	"testing"
	"time"
)

func TestFactoryCanGenerateAndDecodeAToken(t *testing.T) {
	jf := NewJWTFactory(crypt.HS256("test"), 1*time.Second, 2*time.Second, func() time.Time {
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}, func(string) (bool, error) {
		return false, nil
	})

	jwt, err := jf.GenerateToken(crypt.JWTDefaultClaims{Name: "test"})
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
	revocCheckerFn := func(string) (bool, error) {
		return false, nil
	}
	// expire time is 2024-10-04 22:22:27
	// expire in 5s
	jf := NewJWTFactory(crypt.HS256("test"), 5*time.Second, 120*time.Second, timeRefFn, revocCheckerFn)

	jwt1, err := jf.GenerateToken(crypt.JWTDefaultClaims{Name: "test"})
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

func TestFactoryDoesNotRefreshAValidToken(t *testing.T) {
	timeRefFn := func() time.Time {
		// 2024-10-04 22:22:22
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}
	revocCheckerFn := func(string) (bool, error) {
		return false, nil
	}
	// expire time is 2024-10-04 22:22:27
	// expire in 5s
	jf := NewJWTFactory(crypt.HS256("test"), 5*time.Second, 20*time.Second, timeRefFn, revocCheckerFn)
	jwt1, err := jf.GenerateToken(crypt.JWTDefaultClaims{Name: "test"})
	if err != nil {
		t.Fail()
	}

	// Pretty much like TestFactoryCanRefreshAToken test, moving forward in time.
	// Before 2024-10-04 22:22:22, now 2024-10-04 22:22:25
	jf.TimeFn = func() time.Time {
		// 2024-10-04 22:22:25
		return timeRefFn().Add(3 * time.Second)
	}

	trial, err := jf.TryRefresh(jwt1)
	// fail if err
	if err != nil ||
		// should be the same token
		trial.Token != jwt1.Token ||
		// expire date should be no different too
		trial.Claims.Expire != jwt1.Claims.Expire {
		t.Fail()
	}
}

func TestFactoryDoesNotTryToRefreshWayTooOldToken(t *testing.T) {
	timeRefFn := func() time.Time {
		// 2024-10-04 22:22:22
		return time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	}
	revocCheckerFn := func(string) (bool, error) {
		return false, nil
	}
	// expire time is 2024-10-04 22:22:27
	// expire in 5s
	jf := NewJWTFactory(crypt.HS256("test"), 5*time.Second, 20*time.Second, timeRefFn, revocCheckerFn)
	jwt1, err := jf.GenerateToken(crypt.JWTDefaultClaims{Name: "test"})
	if err != nil {
		t.Fail()
	}

	// Pretty much like TestFactoryCanRefreshAToken test, moving forward in time.
	// Before 2024-10-04 22:22:22, now 2024-10-04 22:25:22
	jf.TimeFn = func() time.Time {
		// 2024-10-04 22:25:22
		return timeRefFn().Add(3 * time.Minute)
	}

	trial, err := jf.TryRefresh(jwt1)
	// fail if no err
	if err == nil ||
		// should be an empty token
		trial.Token != "" ||
		err.Error() != TOO_OLD_JWT_ERR {
		t.Fail()
	}
}
