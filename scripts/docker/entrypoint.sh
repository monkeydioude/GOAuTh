#!/bin/bash

PG_PORT=${PG_PORT:-5432}

until pg_isready -h postgres -p ${PG_PORT} -U dev; do
  echo "Waiting for postgres to be ready..."
  sleep 1
done

cat <<EOF > /app/.env
DB_PATH=${DB_PATH:-postgres://dev:dev@postgres:5432/dev_db}
DB_SCHEMA=${DB_SCHEMA:-users}
API_PORT=${API_PORT:-8100}
RPC_PORT=${RPC_PORT:-9100}
EOF

exec "$@"