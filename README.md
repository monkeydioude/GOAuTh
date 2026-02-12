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

> **Security note:** `JWT_SECRET` and `PASSWD_SALT` should be provided via system environment variables or a secrets manager in production. The `.env` file is only suitable for development.

### Defaults

- JWT expiration: **24 hours**
- JWT refresh window: **4 weeks**
- Argon2 parameters: time=3, memory=32 MiB, threads=4, keyLen=32 (RFC draft defaults)
- Minimum password length: **4 characters**
- Login constraint: must be a **valid email address**

## JSON API

All routes are prefixed with `/identity/v1`.

| Method   | Route                          | Description            |
|----------|--------------------------------|------------------------|
| `POST`   | `/identity/v1/auth/signup`     | Create a new user      |
| `PUT`    | `/identity/v1/auth/login`      | Authenticate a user    |
| `GET`    | `/identity/v1/jwt/status`      | Check JWT validity     |
| `PUT`    | `/identity/v1/jwt/refresh`     | Refresh an expired JWT |
| `PUT`    | `/identity/v1/user/password`   | Change password        |
| `PUT`    | `/identity/v1/user/login`      | Change login (email)   |
| `DELETE` | `/identity/v1/user/deactivate` | Soft-delete a user     |
| `GET`    | `/identity/healthcheck`        | Health check           |

All requests are tagged with an `X-Request-ID` header (generated if missing). Authenticated endpoints expect an `Authorization` cookie containing `Bearer {JWT}`.

## gRPC API

Defined in [`proto/rpc_v1.proto`](./proto/rpc_v1.proto).

### Auth Service

| RPC                                | Description           |
|------------------------------------|-----------------------|
| `Signup(UserRequest) → Response`   | Create a new user     |
| `Login(UserRequest) → Response`    | Authenticate a user   |
| `Delete(AuthIdRequest) → Response` | Delete a user by ID   |

### JWT Service

| RPC                        | Description            |
|----------------------------|------------------------|
| `Status(Empty) → Response` | Check JWT validity     |
| `Refresh(Empty) → Response`| Refresh an expired JWT |

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

JWT is passed via a `set-cookie` gRPC metadata entry containing `Authorization=Bearer {JWT}`.

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
}
```

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

No JSON body. Requires an `Authorization` cookie (HTTP) or `set-cookie` metadata (gRPC) containing `Bearer {JWT}`.

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
- **Realm-based namespacing**: users belong to realms, allowing multi-tenant setups.
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
- [`bin/spy-token/`](./bin/spy-token/) — JWT token generator for development/debugging
