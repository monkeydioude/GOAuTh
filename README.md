# GOAuTh
[![Go](https://github.com/monkeydioude/GOAuTh/actions/workflows/go.yml/badge.svg)](https://github.com/monkeydioude/GOAuTh/actions/workflows/go.yml)

Auth provider web-sevice with user creation.

## HOW TO!

The following [JSON API routes](./bin/GOAuTh/api.go) are available (details about the payload will follow):
- /identity/v1/auth/signup (for user creation)
- /identity/v1/auth/login (for user authentification)
- /identity/v1/jwt/status (for jwt health status)
- /identity/v1/jwt/refresh (for jwt refreshing)

The following [RPCs](./proto/rpc_v1.proto) are available:
- Auth.Signup(UserRequest) returns (Response) (for user creation)
- Auth.Login(UserRequest) returns (Response) (for user authentification)
- JWT.Status(Empty) returns (Response) (for jwt health status)
- JWT.Refresh(Empty) returns (Response) (for jwt refreshing)

## CONFIG!

MANDATORY CONFIG ENV VARS:
- DB_PATH=postgres://test:tedt@0.0.0.0:5454/test
- DB_SCHEMA=users

OPTIONAL CONFIG ENV VARS:
- API_PORT=
- RPC_PORT=

DEFAULT COMFIG VARS (will be used if no env var was provided):
- API_PORT: 8100
- RPC_PORT: 9100

__env vars can be provided through a `.env` file, in GOAuTh bin's directory.__

## PAYLOADS!

As a DDDish/Clean Codeish code architecture, each group of api/rpc use the same entity, thus have the same payload.

__API/auth__:
```go
type User struct {
	Login        string         `json:"login"`
	Password     string         `json:"password,omitempty"`
}
```

__RPC/auth__:
```proto
message UserRequest {
    string login = 1;
    string password = 2;
}
```

__API/jwt__ no payload, but a `Authorization {JWT}` _cookie_.

__RPC/jwt__, no payload, but a `set-cookie` _metadata_ containing `Authorization {JWT}`.
