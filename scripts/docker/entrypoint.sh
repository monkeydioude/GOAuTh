#!/bin/bash
DB_PATH=postgres://${PGUSER:-dev}:${PGPASSWORD:-dev}@${PGHOST:-db}:5432/${PGDATABASE:dev_db}
PG_PORT=${PG_PORT:-5432}

until pg_isready -h ${PGHOST} -p ${PG_PORT} -U dev; do
  echo "Waiting for postgres to be ready..."
  sleep 1
done

cat <<EOF > /app/.env
DB_PATH=${DB_PATH:-postgres://dev:dev@db:5432/dev_db}
DB_SCHEMA=${DB_SCHEMA:-users}
API_PORT=${API_PORT:-8100}
RPC_PORT=${RPC_PORT:-9100}
EOF

exec "$@"