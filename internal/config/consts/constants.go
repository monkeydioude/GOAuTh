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
const SetCookie = "set-cookie"

const JWTExpiresIn = 30 * time.Minute
const JWTRefreshesIn = 60 * time.Minute

const BaseAPI_V1 = "/identity/v1"

// Argon2 basic conf
// Should be replaced with actual passed on from the system conf.
// This config is sort of the default argon2 config, recommended by the draft RFC
var Argon2 = crypt.Argon2Bag{
	Time:    3,
	Mem:     32 * 1024,
	Threads: 4,
	KeyLen:  32,
}

const X_REQUEST_ID_LABEL = "X-Request-ID"
const NO_X_REQUEST_ID = "no_x_request_id"

const ERR_TOKEN_EXPIRED = "TokenExpiredError"
const ERR_WRONG_TOKEN_SCHEMA = "TokenSchemaError"
