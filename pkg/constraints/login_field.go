package constraints

import (
	"errors"
	"log"
	"regexp"
)

type EntityField func(string) error

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
