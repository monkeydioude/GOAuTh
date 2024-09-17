package entities

import (
	"testing"
	"time"
)

func TestIsRevoked(t *testing.T) {
	trial := NewDefaultUser()
	timeRef := time.Date(2024, 10, 04, 22, 22, 22, 0, time.UTC)
	// nil RevokedAt
	if trial.IsRevoked(timeRef) == true {
		t.Fail()
	}

	// move revokedAt a bit forward in time, so revokedAt is AFTER
	// the time reference, which means not revoked yet
	t1Ref := timeRef.Add(3 * time.Second)
	trial.RevokedAt = &t1Ref
	if trial.IsRevoked(timeRef) == true {
		t.Fail()
	}

	// opposite as before, we move revokedAt a bit backward in time, so revokedAt is BEFORE
	// the timereference, meaning revoked
	t2Ref := timeRef.Add(-3 * time.Minute)
	trial.RevokedAt = &t2Ref
	if trial.IsRevoked(timeRef) == false {
		t.Fail()
	}
}
