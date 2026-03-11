#!/usr/bin/env bash
# Start a PostgreSQL container for integration tests.
# Usage: source scripts/testdb-start.sh
#        ./scripts/testdb-start.sh  (prints export command at the end)

set -euo pipefail

CONTAINER_NAME="pgbkrs-testdb"
POSTGRES_PORT=15432
POSTGRES_PASSWORD="pgbkrs_test"
POSTGRES_DB="pgbkrs_test"
POSTGRES_USER="pgbkrs"

export TEST_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

# Already running?
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "✓ ${CONTAINER_NAME} is already running"
  echo "export TEST_DATABASE_URL=\"${TEST_DATABASE_URL}\""
  exit 0
fi

# Stopped but exists?
if docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  echo "◆ Restarting existing container ${CONTAINER_NAME}..."
  docker start "${CONTAINER_NAME}" > /dev/null
else
  echo "◆ Starting new PostgreSQL container (port ${POSTGRES_PORT})..."
  docker run -d \
    --name "${CONTAINER_NAME}" \
    -e POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
    -e POSTGRES_USER="${POSTGRES_USER}" \
    -e POSTGRES_DB="${POSTGRES_DB}" \
    -p "${POSTGRES_PORT}:5432" \
    postgres:16-alpine > /dev/null
fi

# Wait for ready
echo -n "  Waiting for PostgreSQL..."
for i in $(seq 1 30); do
  if docker exec "${CONTAINER_NAME}" pg_isready -U "${POSTGRES_USER}" -q 2>/dev/null; then
    echo " ready"
    break
  fi
  echo -n "."
  sleep 1
  if [[ $i -eq 30 ]]; then
    echo " timeout"
    echo "✗ PostgreSQL did not become ready in 30 seconds" >&2
    exit 1
  fi
done

echo "✓ ${CONTAINER_NAME} is ready"
echo ""
echo "export TEST_DATABASE_URL=\"${TEST_DATABASE_URL}\""
