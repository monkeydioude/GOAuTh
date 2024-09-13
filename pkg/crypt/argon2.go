package crypt

import (
	"golang.org/x/crypto/argon2"
)

type Argon2Bag struct {
	Time    uint32
	Mem     uint32
	Threads uint8
	KeyLen  uint32
}

// Argon2KeyHashBytes generates an [argon2](https://en.wikipedia.org/wiki/Argon2) hash.
// argon2params should be set in consideration with memory usage and hash complexity/safety.
// [Don't use that salt on your steak](https://en.wikipedia.org/wiki/Salt_(cryptography))
func Argon2KeyHashBytes(
	bytesToHash []byte,
	argon2params Argon2Bag,
	salt []byte,
) []byte {
	return argon2.Key(
		bytesToHash,
		salt,
		argon2params.Time,
		argon2params.Mem,
		argon2params.Threads,
		argon2params.KeyLen,
	)
}
