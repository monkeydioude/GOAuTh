# GOAuTh

[![Go](https://github.com/monkeydioude/GOAuTh/actions/workflows/go.yml/badge.svg)](https://github.com/monkeydioude/GOAuTh/actions/workflows/go.yml)

Authentication provider web service written in Go. Exposes both a JSON REST API and a gRPC server for user creation, authentication, JWT management, user actions, and realm-based namespacing.

## Table of Contents

- [GOAuTh](#goauth)
  - [Table of Contents](#table-of-contents)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Quick Start](#quick-start)
  - [Configuration](#configuration)
    - [Required](#required)
    - [Optional](#optional)
    - [Defaults](#defaults)
  - [JSON API](#json-api)
  - [gRPC API](#grpc-api)
    - [Auth Service](#auth-service)
    - [JWT Service](#jwt-service)
    - [User Service](#user-service)
    - [UserAction Service](#useraction-service)
  - [Payloads](#payloads)
    - [Auth (Signup / Login)](#auth-signup--login)
    - [Edit User (Password / Login)](#edit-user-password--login)
    - [JWT (Status / Refresh)](#jwt-status--refresh)
    - [User Actions](#user-actions)
  - [Architecture](#architecture)
    - [Key Design Decisions](#key-design-decisions)
  - [Docker](#docker)
    - [Development Database](#development-database)
    - [Production Image](#production-image)
  - [Development](#development)
  - [Testing](#testing)
  - [Makefile Targets](#makefile-targets)
  - [Other Binaries](#other-binaries)

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL (or Docker for a containerized instance)

### Quick Start

```bash
# Install git hooks (run once)
make install

# Start the dev PostgreSQL container and run GOAuTh with hot-reload
make dev
```

GOAuTh will start two servers concurrently:

- **HTTP API** on the configured `API_PORT` (default `:8100`)
- **gRPC server** on the configured `RPC_PORT` (default `:9100`)

Both servers shut down gracefully on `SIGINT` / `SIGTERM`.

## Configuration

Environment variables can be provided through a `.env` file in GOAuTh's working directory (loaded via [godotenv](https://github.com/joho/godotenv)).

### Required

| Variable    | Description                          | Example                                       |
|-------------|--------------------------------------|-----------------------------------------------|
| `DB_PATH`   | PostgreSQL connection string         | `postgres://user:pass@localhost:5432/mydb`    |
| `DB_SCHEMA` | Database schema name                 | `users` (falls back to `public` if empty)     |

### Optional

| Variable                 | Default   | Description                              |
|--------------------------|-----------|------------------------------------------|
| `API_PORT`               | `8100`    | HTTP API listen port                     |
| `RPC_PORT` / `GRPC_PORT` | `9100`    | gRPC listen port                         |
| `JWT_SECRET`             | —         | Secret used for JWT HS256 signing        |
| `PASSWD_SALT`            | —         | Salt used for Argon2 password hashing    |
| `DB_LOG_LEVEL`           | `error`   | GORM log level (`info`, `warn`, `error`) |
| `DB_SLOW_QUERY_LOG_MS`   | `20`      | Slow query threshold in milliseconds     |
| `TRUSTED_PROXIES`        | —         | Comma-separated CIDRs or IPs of the proxies allowed to set `X-Forwarded-For`; empty trusts none |
| `SESSION_TTL_DAYS`       | `30`      | Days a session lives without a refresh; each refresh resets it |
| `SESSION_MAX_ACTIVE`     | `10`      | Active sessions per user; a login beyond it revokes the least recently used one |
| `SESSION_REUSE_GRACE_SECONDS` | `30` | After a refresh, how long the previous refresh token still gets an access token, for refreshes racing each other. A gRPC consumer that doesn't store rotated tokens needs it as long as a session until it does |
| `SESSION_RETENTION_DAYS` | `90` | Days revoked and expired sessions stay listed before `job-session-purge` deletes them |
| `ACCESS_KEY_MAX_ACTIVE` | `20` | Live access keys an account may hold. A realm's `access_key_max_active` may cap lower, never higher |

> **Security note:** `JWT_SECRET` and `PASSWD_SALT` should be provided via system environment variables or a secrets manager in production. The `.env` file is only suitable for development.

### Defaults

- Access token expiration: **1 hour**
- Session (refresh token) lifetime: **30 days**, reset on every refresh
- Argon2 parameters: time=3, memory=32 MiB, threads=4, keyLen=32 (RFC draft defaults)
- Minimum password length: **4 characters**
- Login constraint: must be a **valid email address**

## JSON API

All routes are prefixed with `/identity/v1`.

| Method   | Route                          | Description            |
|----------|--------------------------------|------------------------|
| `POST`   | `/identity/v1/auth/signup`     | Create a new user      |
| `PUT`    | `/identity/v1/auth/login`      | Authenticate a user    |
| `PUT`    | `/identity/v1/auth/logout`     | End the caller's session, named by the `Refresh` cookie or else the access token (even expired), and clear both cookies |
| `GET`    | `/identity/v1/jwt/status`      | Check JWT validity     |
| `PUT`    | `/identity/v1/jwt/refresh`     | Rotate the refresh token, get a new access token |
| `PUT`    | `/identity/v1/user/password`   | Change password        |
| `PUT`    | `/identity/v1/user/login`      | Change login (email)   |
| `DELETE` | `/identity/v1/user/deactivate` | Soft-delete a user     |
| `GET`    | `/identity/v1/sessions`        | The caller's active sessions, most recently used first, with device, IPs, dates and `current` for the calling one; `?include_revoked=true` adds revoked and expired ones |
| `DELETE` | `/identity/v1/sessions/{id}`   | End one of the caller's sessions; `404` when the caller has no such active session |
| `DELETE` | `/identity/v1/sessions`        | End all of the caller's sessions; `?keep_current=true` keeps the calling one |
| `GET`    | `/identity/healthcheck`        | Health check           |

All requests are tagged with an `X-Request-ID` header (generated if missing). Authenticated endpoints expect an `Authorization` cookie containing `Bearer {JWT}`.

## gRPC API

Defined in [`proto/rpc_v1.proto`](./proto/rpc_v1.proto).

### Auth Service

| RPC                                | Description           |
|------------------------------------|-----------------------|
| `Signup(UserRequest) → Response`   | Create a new user     |
| `Login(UserRequest) → Response`    | Authenticate a user   |
| `Delete(AuthIdRequest) → Response` | Close `uid`'s account the way its realm's kind says: soft-deleted, sessions revoked; already gone is a no-op. A service account requires `actor`, who asked as the consumer names them (`422` without) |
| `Logout(LogoutRequest) → Response` | End the calling session, named by the access token in the `Authorization` metadata or by `refresh_token`; an expired or revoked one is a no-op. Without either, ends all of `uid`'s sessions in `realm` (deprecated) |

### JWT Service

| RPC                        | Description            |
|----------------------------|------------------------|
| `Status(Empty) → Response` | Check JWT validity     |
| `Refresh(RefreshIn) → RefreshOut` | Rotate the refresh token, get a new access token |

### User Service

| RPC                                   | Description           |
|---------------------------------------|-----------------------|
| `Deactivate(Empty) → Response`        | Soft-delete a user    |
| `EditUser(EditUserRequest) → Response`| Change login/password |

### UserAction Service

| RPC                                                   | Description                  |
|-------------------------------------------------------|------------------------------|
| `Create(UserActionRequest) → Response`                | Create a user action         |
| `Validate(UserActionValidation) → Response`           | Validate a user action       |
| `Status(UserActionRequest) → UserActionStatusResponse` | Get user action status      |

### Session Service

Acts on the sessions of the user whose access token is in the `Authorization` metadata. A session is one login on a device.

| RPC                                                   | Description                  |
|-------------------------------------------------------|------------------------------|
| `List(ListSessionsRequest) → ListSessionsResponse`    | The user's active sessions, most recently used first, with device, IPs, dates and `current` for the calling one. `include_revoked` adds revoked and expired sessions, with when and why they were revoked |
| `Revoke(RevokeSessionRequest) → Response`             | End one of the user's sessions; `404` when the user has no such active session |
| `RevokeAll(RevokeAllSessionsRequest) → Response`      | End all of the user's sessions, except the calling one with `keep_current` |

JWT is passed via a `set-cookie` gRPC metadata entry containing `Authorization=Bearer {JWT}`.

### Account Service

The accounts a trusted backend manages: those of a `service` realm. People sign up through `Auth` instead. No HTTP route.

| RPC                                                   | Description                  |
|-------------------------------------------------------|------------------------------|
| `Create(CreateAccountRequest) → CreateAccountResponse` | Make an account in `realm` for `login` (a slug: lowercase letters, digits and `:._-`, never an email) on behalf of `actor` (required, free-form, at most 255 characters, stored as `created_by`). The account has no password and cannot log in. `201` with `account_id`, `login`, `realm`, `realm_kind` and `created_at`; `404` unknown realm, `403` a realm whose kind has no such accounts, `422` bad login or actor, `409` login taken in the realm |

### AccessKey Service

Long-lived secrets of the accounts a trusted backend manages. A key is `gak_` followed by 256 random bits (47 characters), shown once at creation and kept only as a SHA-256 hash; a `prefix` of its first characters tells keys apart. Only accounts of a `service` realm hold keys. Every call but `Verify` names the account by `account_id` and `realm`: `404` when the account is not in that realm, `403` when its realm's kind holds no keys, `422` when `actor` (required, at most 255 characters) is missing. Revoked keys are kept forever, with who revoked them and why (`manual`, `account_deleted`).

| RPC                                                   | Description                  |
|-------------------------------------------------------|------------------------------|
| `Create(CreateAccessKeyRequest) → CreateAccessKeyResponse` | Mint a key named `name` (1 to 100 characters, not unique), with an optional `expires_at` (`422` in the past). `201` with `key`, the only time it is returned, and its `info`. `409` when the account holds as many live keys as its realm's `access_key_max_active` or `ACCESS_KEY_MAX_ACTIVE` allows, whichever is lower |
| `List(ListAccessKeysRequest) → ListAccessKeysResponse` | The account's live keys, newest first, never the hash. `include_revoked` adds revoked and expired keys, with when, who and why |
| `Revoke(RevokeAccessKeyRequest) → Response`           | End one key for good, recording `actor`; an expired key can still be revoked. `404` when the account has no such live key |
| `Verify(VerifyAccessKeyRequest) → VerifyAccessKeyResponse` | Say whose `key` is: `200` with `key_id`, `account_id`, `login`, `realm`, `realm_kind` and `expires_at` (unset when it never expires). `401 InvalidKey` for a key that is malformed, unknown, revoked or expired, or whose account is deleted or revoked or whose realm is gone, without telling which. Takes no `account_id`: the key names itself. Not cached, so a revocation shows at once. Notes the key as used (`last_used_at`), at most once a minute |

`Auth.Delete` on a service account revokes its keys in the same transaction (`account_deleted`, by the request's `actor`).

## Payloads

### Auth (Signup / Login)

**JSON:**

```json
{
  "login": "user@example.com",
  "password": "s3cure!",
  "realm_name": "my-realm"
}
```

**Protobuf:**

```protobuf
message UserRequest {
    string login = 1;
    string password = 2;
    string realm = 3;
    optional int64 access_expires_in_seconds = 4;
    // ignored: a refresh token lives as long as its session (SESSION_TTL_DAYS)
    optional int64 refresh_expires_in_seconds = 5 [deprecated = true];
    ClientInfo client = 6;
}

message ClientInfo {
    string ip = 1;
    string user_agent = 2;
}
```

gRPC callers are usually backends, so goauth would otherwise see their address, not the user's. They send the end user's IP and user agent in `client`, on `UserRequest` and `RefreshIn`. Over HTTP, goauth reads the `User-Agent` header and the client IP, honoring `X-Forwarded-For` only from `TRUSTED_PROXIES`.

### Edit User (Password / Login)

**JSON (password change):**

```json
{
  "password": "current-password",
  "new_password": "new-s3cure!"
}
```

**JSON (login change):**

```json
{
  "password": "current-password",
  "login": "old@example.com",
  "new_login": "new@example.com"
}
```

**Protobuf:**

```protobuf
message EditUserRequest {
    string new_login = 1;
    string new_password = 2;
    string password = 3;
}
```

### JWT (Status / Refresh)

No JSON body. Status reads the `Authorization` cookie (HTTP) or `set-cookie` metadata (gRPC) containing `Bearer {JWT}`. Refresh reads the `Refresh` cookie, or `refreshToken` in `RefreshIn` over gRPC. Status also checks, in one query, that the token's session is still active and its user not deactivated, so a revoked session loses access on its next request.

Each refresh rotates the session's refresh token: store the new one, the old one stops working. If two refreshes race with the same token, the second gets a new access token but no refresh token (`RefreshOut.refreshToken` empty, no `Refresh` cookie), so keep the one the first returned. Presenting an older refresh token after `SESSION_REUSE_GRACE_SECONDS` revokes the session (`TokenReused`). A refresh only touches the session its token belongs to.

Account events revoke sessions too: a password change revokes every session but the one that made it (`password_changed`), a password reset every session (`password_reset`), and deactivation or deletion every session (`account_deactivated`).

### User Actions

```protobuf
message UserActionRequest {
    string login = 1;
    string realm = 2;
    string action = 3;
    string data = 4;
}

message UserActionValidation {
    string realm = 1;
    string data = 2;
    string against = 3;
}
```

## Architecture

```
bin/GOAuTh/          → Main binary (HTTP + gRPC servers)
internal/
  api/handlers/      → HTTP route handlers (auth, jwt, user)
  config/boot/       → Bootstrap (DB, JWT factory, user params)
  config/consts/     → Environment variable names, defaults, error codes
  config/middleware/  → Request logging, X-Request-ID (HTTP & gRPC)
  domain/entities/   → GORM models (User, Realm, UserAction), constraints
  domain/models/     → UsersParams (salt, Argon2 config, constraints)
  domain/services/   → Business logic (auth, jwt, user, user actions)
pkg/
  crypt/             → Argon2 hashing, JWT encode/decode (HS256)
  data_types/        → Utility types (ptr, slice, tuple)
  domain/entities/   → Generic interfaces (User, JWT)
  errors/            → Error type with HTTP status mapping
  grpc/v1/           → gRPC server implementations + generated code
  http/              → Middleware, JSON request/response, RPC cookie helpers
  plugins/           → Event-based plugin system (before/after hooks)
plugins/             → Loadable plugin files (e.g. heyo broker integration)
proto/               → Protocol Buffer definitions
```

### Key Design Decisions

- **DDD-ish / Clean Architecture**: domain entities and services are separated from transport (HTTP/gRPC) handlers.
- **Realm-based namespacing**: users belong to realms, allowing multi-tenant setups. A login is unique within its realm, so the same email can sign up in two realms.
- **Realm kinds**: a realm is of kind `human` (the default) or `service`. A service realm holds accounts that are not people, so it refuses signup, password and login changes and user actions with `403 ForbiddenByRealmKind`, and login with `401 InvalidCredentials`. The kind is set by `bin/client realm create -kind=…` and never changes. Its accounts are made by the `Account` gRPC service and closed by `Auth.Delete`, both naming who asked, and they alone hold access keys (`AccessKey` service).
- **Dual transport**: the same service layer is exposed over both HTTP and gRPC.
- **Plugin system**: event hooks (`OnUserCreation`, etc.) with configurable timeouts for extensibility.
- **Graceful shutdown**: `oklog/run` coordinates concurrent servers and OS signal handling.

## Docker

### Development Database

```bash
docker compose up -d   # Starts PostgreSQL (dev/dev/dev_db) on port 5432
```

### Production Image

```bash
docker build -t drannoc/goauth .
```

The multi-stage Dockerfile builds `GOAuTh` and the `client` binary, then runs via an entrypoint script that:
1. Waits for PostgreSQL readiness
2. Writes a `.env` file from container environment variables
3. Starts GOAuTh (or a custom command)

## Development

```bash
# Start containers + hot-reload server (uses gow)
make dev

# Regenerate protobuf code (Go + Rust)
make proto

# Access the dev database via psql
make dpsql
```

## Testing

```bash
# Unit tests only (internal, pkg, plugins)
make unit-test

# Full test suite (spins up a test DB container, runs all tests)
make test

# Manually manage the test database container
make run-test-db
make stop-test-db
```

## Makefile Targets

| Target          | Description                                           |
|-----------------|-------------------------------------------------------|
| `help`          | Show available targets                                |
| `install`       | Install git pre-commit hook                           |
| `all_up`        | Start all Docker Compose services                     |
| `db_layout`     | Start DB container + import initial SQL               |
| `dev`           | Start Docker + hot-reload GOAuTh with `gow`           |
| `unit-test`     | Run unit tests                                        |
| `test`          | Run full test suite (with ephemeral test DB)          |
| `run-test-db`   | Start a standalone test PostgreSQL container          |
| `stop-test-db`  | Stop the standalone test PostgreSQL container         |
| `proto-go`      | Regenerate Go protobuf/gRPC code                      |
| `proto-rust`    | Build Rust protobuf client                            |
| `proto`         | Regenerate both Go and Rust protobuf code             |
| `docker-build`  | Build and push the Docker image                       |
| `dpsql`         | Open a psql shell to the dev database                 |

## Other Binaries

Additional utility binaries live in `bin/`. Each has its own README:

- [`bin/client/`](./bin/client/) — CLI client for interacting with GOAuTh via API or gRPC
- [`bin/min-grpc-server/`](./bin/min-grpc-server/) — Minimal gRPC-only server (no HTTP)
- [`bin/job-session-purge/`](./bin/job-session-purge/) — Daily job deleting sessions ended more than `SESSION_RETENTION_DAYS` ago
- [`bin/spy-token/`](./bin/spy-token/) — JWT token generator for development/debugging
