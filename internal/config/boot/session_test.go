package boot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionBootDefaults(t *testing.T) {
	trial, err := SessionBoot()

	assert.NoError(t, err)
	assert.Equal(t, 30*24*time.Hour, trial.TTL())
	assert.Equal(t, 10, trial.MaxActive)
}

func TestSessionBootReadsTheEnv(t *testing.T) {
	t.Setenv("SESSION_TTL_DAYS", "7")
	t.Setenv("SESSION_MAX_ACTIVE", "3")

	trial, err := SessionBoot()

	assert.NoError(t, err)
	assert.Equal(t, 7*24*time.Hour, trial.TTL())
	assert.Equal(t, 3, trial.MaxActive)
}

func TestSessionBootRefusesLessThanOne(t *testing.T) {
	t.Setenv("SESSION_MAX_ACTIVE", "0")

	_, err := SessionBoot()

	assert.Error(t, err)
}

func TestRefreshTokensLiveAsLongAsTheirSession(t *testing.T) {
	_, rtf := JwtFactoryBoot(nil, 30*24*time.Hour)

	assert.Equal(t, 30*24*time.Hour, rtf.ExpiresIn)
}
