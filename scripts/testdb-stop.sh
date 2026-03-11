#!/usr/bin/env bash
# Stop and remove the PostgreSQL test container.

CONTAINER_NAME="pgbkrs-testdb"

if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
  docker stop "${CONTAINER_NAME}" > /dev/null
  docker rm "${CONTAINER_NAME}" > /dev/null
  echo "✓ ${CONTAINER_NAME} stopped and removed"
else
  echo "○ ${CONTAINER_NAME} is not running"
fi
