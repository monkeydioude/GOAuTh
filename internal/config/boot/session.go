package boot

import (
	"errors"
	"time"

	"github.com/calqs/gopkg/env"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/timed"
)

type SessionEnv struct {
	TTLDays           int `env:"SESSION_TTL_DAYS,?30"`
	MaxActive         int `env:"SESSION_MAX_ACTIVE,?10"`
	ReuseGraceSeconds int `env:"SESSION_REUSE_GRACE_SECONDS,?30"`
}

// TTL is how long a session lives without a refresh.
func (s SessionEnv) TTL() time.Duration {
	return timed.Days(s.TTLDays)
}

// ReuseGrace is how long after a rotation the previous refresh token still
// gets an access token, for refreshes racing each other.
func (s SessionEnv) ReuseGrace() time.Duration {
	return timed.Seconds(s.ReuseGraceSeconds)
}

// SessionBoot reads how long a session lives without a refresh, how many
// active sessions a user may have, and the grace window for racing refreshes.
func SessionBoot() (SessionEnv, error) {
	config, err := env.ParseEnv[SessionEnv]()
	if err != nil {
		return SessionEnv{}, err
	}
	if config.TTLDays < 1 || config.MaxActive < 1 {
		return SessionEnv{}, errors.New("SESSION_TTL_DAYS and SESSION_MAX_ACTIVE must be at least 1")
	}
	if config.ReuseGraceSeconds < 0 {
		return SessionEnv{}, errors.New("SESSION_REUSE_GRACE_SECONDS cannot be negative")
	}
	return config, nil
}
