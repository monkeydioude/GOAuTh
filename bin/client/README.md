# client

CLI client for interacting with a running GOAuTh instance. Supports both the JSON REST API and gRPC transports.

## Build

```bash
go build -C bin/client -o client
```

## Usage

```bash
./client -method=<api|rpc> <service> <action> [extra args...]
./client help
```

### Flags

| Flag     | Default | Description                    |
|----------|---------|--------------------------------|
| `-method`| `api`   | Transport method: `api` or `rpc` |

### Services & Actions

| Service  | Action       | Transport  | Description                          |
|----------|-------------|------------|--------------------------------------|
| `auth`   | `login`     | api, rpc   | Authenticate a user                  |
| `auth`   | `signup`    | api, rpc   | Create a new user                    |
| `jwt`    | `status`    | api, rpc   | Check JWT validity                   |
| `jwt`    | `refresh`   | api, rpc   | Refresh a JWT                        |
| `user`   | `password`  | api        | Change user password                 |
| `user`   | `login`     | api        | Change user login (email)            |
| `user`   | `deactivate`| api        | Soft-delete a user                   |
| `user`   | `change_user`| api, rpc  | Edit user (rpc: login/password)      |
| `realm`  | `create`    | api, rpc   | Create a new realm                   |
| `realm`  | `view`      | api        | List all realms                      |
| `action` | `create`    | rpc only   | Create a user action                 |
| `action` | `validate`  | rpc only   | Validate a user action               |
| `action` | `status`    | rpc only   | Get user action status               |

## Environment Variables

### Authentication

| Variable          | Description                                      |
|-------------------|--------------------------------------------------|
| `CLIENT_LOGIN`    | Login (email) for auth operations                |
| `CLIENT_PASSWORD` | Password for auth operations                     |
| `CLIENT_REALM`    | Realm name                                       |
| `CLIENT_JWT`      | JWT token for authenticated requests             |

### Connection

| Variable   | Description                                           |
|------------|-------------------------------------------------------|
| `API_URL`  | Base URL of the GOAuTh HTTP API (e.g. `http://localhost:8100`) |

The gRPC client connects to `[::]:9100` by default.

### Password / Login Change

| Variable        | Description              |
|-----------------|--------------------------|
| `OLD_PASSWORD`  | Current password         |
| `NEW_PASSWORD`  | New password             |
| `OLD_LOGIN`     | Current login (email)    |
| `NEW_LOGIN`     | New login (email)        |

### User Action Validation (gRPC)

| Variable                          | Description                     |
|-----------------------------------|---------------------------------|
| `USER_ACTION_VALIDATION_DATA`     | Data to validate                |
| `USER_ACTION_VALIDATION_AGAINST`  | Value to validate against       |

### Realm / Action Extra Args

For `realm create`, extra positional args after the action are:

```
./client -method=api realm create <allow_new_user=0|1> <name> [description]
```

For `action create` and `action status` (rpc), extra positional args are:

```
./client -method=rpc action create <login> <realm> <action> <data>
./client -method=rpc action status <login> <realm> <action> <data>
```

## Examples

```bash
# Signup via API
CLIENT_LOGIN=user@example.com CLIENT_PASSWORD=s3cure! CLIENT_REALM=default \
  ./client -method=api auth signup

# Login via gRPC
CLIENT_LOGIN=user@example.com CLIENT_PASSWORD=s3cure! CLIENT_REALM=default \
  ./client -method=rpc auth login

# Check JWT status via API
CLIENT_JWT=eyJhbGci... API_URL=http://localhost:8100 \
  ./client -method=api jwt status

# Create a realm
./client -method=api realm create 1 my-realm "My realm description"

# Create a user action via gRPC
./client -method=rpc action create user@example.com default reset-password some-data
```

## Database

The `realm` commands (`create`, `view`) connect directly to the database using the standard `DB_PATH` and `DB_SCHEMA` environment variables (loaded from `.env`).
