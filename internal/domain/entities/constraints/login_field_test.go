package constraints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIMatchACorrectStrings(t *testing.T) {
	trials := []string{"te.st@test.com", "a@b.fr", "in...gouda.wetrustvery_very+veryvery.very@veeeeeeery.hard.realtalk.co.jp"}
	for _, trial := range trials {
		assert.NoError(t, EmailConstraint(trial, nil))
	}
}

func TestIFailOnMalformatedStrings(t *testing.T) {
	trials := []string{"test.com", "a@", "ingoudawetrustveryveryveryvery.very_veeeeeeery.hard.realtalk.co.jp"}
	for _, trial := range trials {
		assert.Error(t, EmailConstraint(trial, nil))
	}
}

func TestIFailOnSameLogin(t *testing.T) {
	old := "passwd"
	trial := old
	assert.Error(t, EmailConstraint(trial, &old))
}
