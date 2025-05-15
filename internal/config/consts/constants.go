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

const JWTExpiresIn = 1 * 24 * time.Hour
const JWTRefreshesIn = 4 * 7 * 24 * time.Hour

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

const ERR_TOKEN_MISSING_PARAMS = "TokenMissingParameters"
const ERR_TOKEN_EXPIRED = "TokenExpiredError"
const ERR_WRONG_TOKEN_SCHEMA = "TokenSchemaError"
const ERR_MISSING_TOKEN = "MissingToken"
const ERR_INVALID_CREDENTIALS = "InvalidCredentials"
const ERR_USER_ALREADY_EXIST = "UserAlreadyExists"
const ERR_INTERNAL_ERROR = "InternalError"
const ERR_INVALID_INPUT_PASSWORD = "InvalidInputPassword"
const ERR_SAME_INPUT_PASSWORD = "SameInputPassword"
const ERR_INVALID_INPUT_LOGIN = "InvalidInputLogin"
const ERR_PASSWORD_VALIDATION = "PasswordValidationError"
const CONSTRAINT_PASSWORD_MIN_SIZE = 4
