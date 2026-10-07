# job-session-purge

Deletes for good the sessions revoked or expired more than `SESSION_RETENTION_DAYS` ago, then exits. Until then, they stay listed in the users' session history.

Schedule it once a day (cron, Kubernetes CronJob...). It runs no migrations: the main GOAuTh binary must have created the `sessions` table. Running it more often, or on several hosts at once, is harmless.

## Build

```bash
go build -C bin/job-session-purge -o job-session-purge
```

`Dockerfile.prod` also builds it into the production image, next to `GOAuTh`.

## Usage

```bash
./job-session-purge
```

Logs how many sessions it deleted. Exits non-zero when it cannot reach the database or the purge fails.

## Configuration

Loaded from environment variables or a `.env` file.

| Variable                 | Default  | Description                                                  |
|--------------------------|----------|--------------------------------------------------------------|
| `DB_PATH`                | —        | PostgreSQL connection string                                 |
| `DB_SCHEMA`              | `public` | Database schema                                              |
| `SESSION_RETENTION_DAYS` | `90`     | Days revoked and expired sessions are kept before deletion   |
