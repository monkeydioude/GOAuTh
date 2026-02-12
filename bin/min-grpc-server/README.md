# min-grpc-server

Minimal gRPC-only authentication server. Exposes the same Auth, JWT, and User gRPC services as the main GOAuTh binary, but without the HTTP REST API. Useful for lightweight deployments, integration testing, or environments where only gRPC is needed.

## Build

```bash
go build -C bin/min-grpc-server -o min-grpc-server
```

## Usage

```bash
./min-grpc-server
```

The server listens on the configured gRPC port (default `:9100`) and shuts down gracefully on `SIGINT` / `SIGTERM`.

## Configuration

Loaded from environment variables or a `.env` file (required — the server panics if `.env` is not found).

| Variable                 | Default                                    | Description                          |
|--------------------------|--------------------------------------------|--------------------------------------|
| `DB_PATH`                | `postgres://user:pass@localhost:5432/mydb` | PostgreSQL connection string         |
| `DB_SCHEMA`              | `public`                                   | Database schema                      |
| `RPC_PORT` / `GRPC_PORT` | `9100`                                     | gRPC listen port                     |
| `JWT_SECRET`             | —                                          | JWT HS256 signing secret             |
| `PASSWD_SALT`            | —                                          | Argon2 password hashing salt         |

## Registered gRPC Services

| Service | RPCs                       |
|---------|----------------------------|
| Auth    | `Signup`, `Login`          |
| JWT     | `Status`, `Refresh`        |
| User    | `Deactivate`, `EditUser`   |

> **Note:** Unlike the full GOAuTh binary, this server does **not** register the `UserAction` service.

## Differences from GOAuTh

- No HTTP API — gRPC only.
- No `UserAction` gRPC service.
- Uses a simplified bootstrap that does not connect to a database or set up a JWT factory at startup — the `Layout` is initialized with only `UserParams` and no DB/JWT bindings. This makes it suitable for scenarios where the database is optional or managed externally.
- Plugin system is disabled (commented out).

## Middleware

- `GRPCLogRequest` — Logs each incoming gRPC request.
- `GRPXRequestID` — Injects/propagates `X-Request-ID` on each call.
