# dev

A developer's shortcuts against the goauth database. It talks to PostgreSQL directly, not to a running GOAuTh, and runs no migrations: the main GOAuTh binary must have created the tables.

## Build

```bash
go build -C bin/dev -o dev
```

Or run it in place with `go run ./bin/dev <args>`.

## Usage

### access-key create

```bash
./dev access-key create <realm> <login>
```

Mints an access key that never expires for the account `<login>` of `<realm>`, named after the login. When the realm has no such account yet, it creates one. The realm must be of kind `service`: a `human` realm refuses both the account and the key with `ForbiddenByRealmKind`.

It prints the key, which is shown this once and only kept hashed, followed by its `key_id` and the `account_id`. Both the account and the key are recorded as created by `cli:<your OS username>`.

It fails with `AccessKeyLimitReached` when the account already holds as many live keys as `ACCESS_KEY_MAX_ACTIVE`, or the realm's lower cap, allows.

## Configuration

Loaded from environment variables or a `.env` file.

| Variable                | Default  | Description                         |
|-------------------------|----------|-------------------------------------|
| `DB_PATH`               | —        | PostgreSQL connection string        |
| `DB_SCHEMA`             | `public` | Database schema                     |
| `ACCESS_KEY_MAX_ACTIVE` | `20`     | Live access keys an account may hold |
