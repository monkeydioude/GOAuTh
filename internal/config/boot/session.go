package boot

import (
	"errors"
	"time"

	"github.com/calqs/gopkg/env"
	"github.com/monkeydioude/goauth/v2/pkg/data_types/timed"
)

type SessionEnv struct {
	TTLDays   int `env:"SESSION_TTL_DAYS,?30"`
	MaxActive int `env:"SESSION_MAX_ACTIVE,?10"`
}

// TTL is how long a session lives without a refresh.
func (s SessionEnv) TTL() time.Duration {
	return timed.Days(s.TTLDays)
}

// SessionBoot reads how long a session lives without a refresh,
// and how many active sessions a user may have.
func SessionBoot() (SessionEnv, error) {
	config, err := env.ParseEnv[SessionEnv]()
	if err != nil {
		return SessionEnv{}, err
	}
	if config.TTLDays < 1 || config.MaxActive < 1 {
		return SessionEnv{}, errors.New("SESSION_TTL_DAYS and SESSION_MAX_ACTIVE must be at least 1")
	}
	return config, nil
}
