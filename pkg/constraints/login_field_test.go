package constraints

import "testing"

func TestIMatchACorrectStrings(t *testing.T) {
	trials := []string{"te.st@test.com", "a@b.fr", "in...gouda.wetrustvery_very+veryvery.very@veeeeeeery.hard.realtalk.co.jp"}
	for _, trial := range trials {
		if EmailConstraint(trial) != nil {
			t.Fail()
		}
	}
}

func TestIFailOnMalformatedStrings(t *testing.T) {
	trials := []string{"test.com", "a@", "ingoudawetrustveryveryveryvery.very_veeeeeeery.hard.realtalk.co.jp"}
	for _, trial := range trials {
		if EmailConstraint(trial) == nil {
			t.Fail()
		}
	}
}
