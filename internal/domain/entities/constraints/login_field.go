package constraints

import (
	"errors"
	"log"
	"regexp"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
)

var slug = regexp.MustCompile(`^[a-z0-9][a-z0-9:._-]{0,127}$`)

// SlugConstraint is the login of an account that is not a person: lowercase
// letters, digits and :._- (1 to 128 characters), so never an email.
func SlugConstraint(login string, _ *string) error {
	if !slug.MatchString(login) {
		return errors.New(consts.ERR_INVALID_INPUT_LOGIN)
	}
	return nil
}

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
