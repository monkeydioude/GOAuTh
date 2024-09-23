package models

import (
	"GOAuTh/pkg/crypt"
)

// UsersParams defines a struct of config that should be
// passed to any user entity. As early as possible.
type UsersParams struct {
	PasswdSalt   []byte
	Argon2params crypt.Argon2Bag
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
