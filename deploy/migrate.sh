#!/bin/bash
set -e

CONTAINER="fqw-postgres"
BACKEND_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$(dirname "$0")/.env"

if [ -f "$ENV_FILE" ]; then
  export $(grep -v '^#' "$ENV_FILE" | grep -E '^PG_(USER|DB)' | xargs)
fi

PG_USER="${PG_USER:-postgres}"
PG_DB="${PG_DB:-fqw}"

run_psql() {
  docker exec "$CONTAINER" psql -U "$PG_USER" -d "$PG_DB" "$@"
}

run_psql -c "CREATE TABLE IF NOT EXISTS schema_migrations (
  name       TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ DEFAULT NOW()
);"

apply_service() {
  local service="$1"
  local dir="$BACKEND_DIR/$service/migrations"

  [ -d "$dir" ] || return 0

  for file in $(ls "$dir"/*.sql 2>/dev/null | sort); do
    local key="$service/$(basename "$file")"
    local applied
    applied=$(run_psql -tAc "SELECT 1 FROM schema_migrations WHERE name='$key'" 2>/dev/null | tr -d '[:space:]')
    if [ "$applied" = "1" ]; then
      echo "  skip : $key"
    else
      echo "  apply: $key"
      docker exec -i "$CONTAINER" psql -U "$PG_USER" -d "$PG_DB" < "$file"
      run_psql -c "INSERT INTO schema_migrations (name) VALUES ('$key');"
    fi
  done
}

echo "=== Migrations ==="
apply_service "auth-service"
apply_service "manage-service"
apply_service "statistics-service"
echo "=== Done ==="
