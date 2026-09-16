#!/usr/bin/env bash
# Run internal/dbtest against a throwaway PostgreSQL, so the tier is runnable off
# CI. `go test ./...` discards a passing package's output, so a tier that skipped
# every case is invisible without this (#2255).
set -euo pipefail

image="postgres:16-bookworm@sha256:60f4761b9035e0b8d5218f701a8c3382f641bf12b1604822574cf5be3baeb537"
container="verge-dbtest"
# Not 5432: a developer's own PostgreSQL, and the compose stack's, already hold it.
port="${VERGE_DBTEST_PORT:-5442}"
password="dbtest-only-password"

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
log="$(mktemp)"

cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -f "$log"
}

if ! docker info >/dev/null 2>&1; then
  echo "error: no reachable Docker daemon, so this script cannot start a database" >&2
  rm -f "$log"
  exit 2
fi

# A container left behind by an interrupted run holds the port and serves stale rows.
docker rm -f "$container" >/dev/null 2>&1 || true
trap cleanup EXIT

docker run -d --rm --name "$container" \
  -e POSTGRES_USER=verge \
  -e POSTGRES_PASSWORD="$password" \
  -e POSTGRES_DB=verge \
  -p "127.0.0.1:$port:5432" \
  "$image" >/dev/null

echo "waiting for postgres on 127.0.0.1:$port"
ready=0
for _ in $(seq 1 60); do
  if docker exec "$container" pg_isready -U verge -q 2>/dev/null; then
    ready=1
    break
  fi
  sleep 1
done
if [ "$ready" -ne 1 ]; then
  echo "error: postgres did not become ready" >&2
  docker logs "$container" >&2 || true
  exit 1
fi

export VERGE_TEST_DATABASE_URL="postgres://verge:$password@127.0.0.1:$port/verge?sslmode=disable"

set +e
(cd "$repo" && go test ./internal/dbtest/... -count=1 -v "$@") 2>&1 | tee "$log"
status="${PIPESTATUS[0]}"
set -e

if ! "$repo/scripts/assert-dbtest-ran.sh" "$log"; then
  status=1
fi
exit "$status"
