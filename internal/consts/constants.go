package consts

import (
	"GOAuTh/pkg/crypt"
	"time"
)

// Env labels
const DB_PATH = "DB_PATH"
const DB_USER = "DB_USER"
const DB_PASSWD = "DB_PASSWD"
const DB_SCHEMA = "DB_SCHEMA"
const JWT_SECRET = "JWT_SECRET"
const PASSWD_SALT = "PASSWD_SALT"

const AuthorizationCookie = "Authorization"

const JWTExpiresIn = 5 * time.Second
const JWTRefreshesIn = 10 * time.Second

// Argon2 basic conf
// Should be replaced with actual passed on from the system conf.
// This config is sort of the default argon2 config, recommended by the draft RFC
var Argon2 = crypt.Argon2Bag{
	Time:    3,
	Mem:     32 * 1024,
	Threads: 4,
	KeyLen:  32,
}
