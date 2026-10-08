package boot

import (
	"errors"

	"github.com/calqs/gopkg/env"
)

type AccessKeyEnv struct {
	// MaxActive is the live access keys an account may hold. A realm may set
	// a lower cap, never a higher one.
	MaxActive int `env:"ACCESS_KEY_MAX_ACTIVE,?20"`
}

// AccessKeyBoot reads how many live access keys an account may hold.
func AccessKeyBoot() (AccessKeyEnv, error) {
	config, err := env.ParseEnv[AccessKeyEnv]()
	if err != nil {
		return AccessKeyEnv{}, err
	}
	if config.MaxActive < 1 {
		return AccessKeyEnv{}, errors.New("ACCESS_KEY_MAX_ACTIVE must be at least 1")
	}
	return config, nil
}
