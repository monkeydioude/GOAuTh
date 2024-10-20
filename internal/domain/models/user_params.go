package models

import (
	"GOAuTh/internal/domain/entities/constraints"
	"GOAuTh/pkg/crypt"
)

// UsersParams defines a struct of config that should be
// passed to any user entity. As early as possible.
type UsersParams struct {
	PasswdSalt          []byte
	Argon2params        crypt.Argon2Bag
	LoginConstraints    []constraints.LoginConstraint
	PasswordConstraints []constraints.PasswordConstraint
}

// GetPasswordSalt returns the salt that is gonna be used by the argon2
// password gen algo.
func (u UsersParams) GetPasswordSalt() []byte {
	return u.PasswdSalt
}

// GetArgon2Params defines how the argon2 params itself should be used.
// Those params tweak the speed and the safety of the generated hash.
// Argon2 being the most secure hashing algo at the time of writing,
// modests parameters will generate a safe, but a "weak" hash in an Argon2 context.
func (u UsersParams) GetArgon2Params() crypt.Argon2Bag {
	return u.Argon2params
}

// AssertLogin will run each specified login constraint
// against a login.
func (u *UsersParams) AssertLogin(login string, old_login *string) error {
	for _, c := range u.LoginConstraints {
		if err := c(login, old_login); err != nil {
			return err
		}
	}
	return nil
}

// AssertPassword will run each specified password constraint
// against a password.
func (u *UsersParams) AssertPassword(password string, old_password *string) error {
	for _, c := range u.PasswordConstraints {
		if err := c(password, old_password); err != nil {
			return err
		}
	}
	return nil
}

// AssertAllConstraints = AssertLogin + AssertPassword
func (u *UsersParams) AssertAllConstraints(login string, old_login *string, password string, old_password *string) error {
	if err := u.AssertLogin(login, old_login); err != nil {
		return err
	}
	return u.AssertPassword(password, old_password)
}
