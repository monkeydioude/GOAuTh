package constraints

import (
	"errors"
	"log"
	"regexp"

	"github.com/monkeydioude/goauth/internal/config/consts"
)

// EmailConstraint is a simple and basic email format tester
func EmailConstraint(email string, _ *string) error {
	matched, err := regexp.Match("^.*@[^.]+..+$", []byte(email))
	if err != nil {
		log.Printf("[WARN] %s\n", err)
		return errors.New(consts.ERR_PASSWORD_VALIDATION)
	}
	if !matched {
		return errors.New(consts.ERR_INVALID_INPUT_LOGIN)
	}
	return nil
}
