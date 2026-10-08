package constraints

import (
	"strings"
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

func TestSlugConstraintAcceptsServiceLogins(t *testing.T) {
	trials := []string{"sb:org:42", "bots", "a", "org-1.prod_eu", "a" + strings.Repeat("0", 127)}
	for _, trial := range trials {
		assert.NoError(t, SlugConstraint(trial, nil), trial)
	}
}

func TestSlugConstraintRefusesEmailsAndTheRest(t *testing.T) {
	trials := []string{"", "Bots", "sb@org.com", ":leading", "-leading", "with space", "a" + strings.Repeat("0", 128)}
	for _, trial := range trials {
		assert.Error(t, SlugConstraint(trial, nil), trial)
	}
}

func TestIFailOnSameLogin(t *testing.T) {
	old := "passwd"
	trial := old
	assert.Error(t, EmailConstraint(trial, &old))
}
