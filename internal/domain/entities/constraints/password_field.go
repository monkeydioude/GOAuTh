package constraints

import (
	"errors"

	"github.com/monkeydioude/goauth/internal/config/consts"
)

// PasswordSafetyConstraint is a simple and basic password format tester
func PasswordSafetyConstraint(password string, oldPassword *string) error {
	if len(password) < consts.CONSTRAINT_PASSWORD_MIN_SIZE {
		return errors.New(consts.ERR_INVALID_INPUT_PASSWORD)
	}
	if oldPassword != nil && *oldPassword == password {
		return errors.New(consts.ERR_SAME_INPUT_PASSWORD)
	}
	return nil
}
