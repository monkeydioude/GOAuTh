package constraints

import (
	"errors"
	"log"
	"regexp"
)

type EntityField func(string) error

// EmailConstraint is a simple and basic email format tester
func EmailConstraint(email string) error {
	matched, err := regexp.Match("^.*@[^.]+..+$", []byte(email))
	if err != nil {
		log.Printf("[WARN] %s\n", err)
		return err
	}
	if !matched {
		return errors.New("format should be ^.*@[^.]+..+$")
	}
	return nil
}
