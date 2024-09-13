package crypt

import (
	"encoding/base64"
)

func HashPassword(
	passwd string,
	argon2Bag Argon2Bag,
	salt []byte,
) string {
	return base64.
		StdEncoding.
		EncodeToString(
			Argon2KeyHashBytes(
				[]byte(passwd),
				argon2Bag,
				salt,
			),
		)
}
