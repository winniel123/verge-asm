#!/usr/bin/env bash
# run.sh — local advisory driver for the /inventory pixel-parity harness (#526).
#
# Orchestrates the whole loop inside the pinned canonical containers so a human or
# CI can reproduce it:
#   1. render-goldens (golang image)   -> design-system/goldens/inventory.html
#   2. capture --write-goldens (pinned Playwright, file://) -> the 10 golden PNGs
#      (skipped unless GOLDENS=write, so a normal run diffs against committed PNGs)
#   3. Postgres (pinned) + web (golang image): migrate + seed fixtures + serve
#   4. capture --mode candidate --advisory (pinned Playwright, on the same docker
#      network) -> per-state-theme differing-pixel %
#
# Repo-owned harness — not a design-owned artifact. Everything canonical runs in a
# pinned-by-digest image. On Windows Git Bash, MSYS_NO_PATHCONV=1 stops path mangling.
#
# Usage:
#   design-system/verify/harness/run.sh              # diff candidate vs committed goldens
#   GOLDENS=write design-system/verify/harness/run.sh # (re)materialize the golden PNGs first
#
# Env knobs: KEEP=1 leaves the pg/web containers + network up for inspection.
set -euo pipefail
export MSYS_NO_PATHCONV=1

# repo root = harness -> verify -> design-system -> repo
REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$REPO"

PW_IMAGE="mcr.microsoft.com/playwright:v1.46.0-jammy@sha256:860c541d62e212fa2d857afca98730dad12b641f941b9b5ed892e379e9e121bb"
PG_IMAGE="postgres:16-bookworm@sha256:60f4761b9035e0b8d5218f701a8c3382f641bf12b1604822574cf5be3baeb537"
GO_IMAGE="golang:1.25-bookworm"

NET="verge-verify-net"
PG="verge-verify-pg"
WEB="verge-verify-web"
NPM_VOL="verge-verify-npm"       # node_modules (pixelmatch/pngjs/playwright) — persisted across runs
BIN_VOL="verge-verify-bin"       # built web binary
DBURL="postgres://verge:verge@${PG}:5432/verge?sslmode=disable"

GO_CACHE=(-v verge-gocache:/root/.cache/go-build -v verge-gomod:/go/pkg/mod)
HARNESS_MNT=(-v "$REPO":/src -w /src/design-system/verify/harness -v "${NPM_VOL}:/src/design-system/verify/harness/node_modules" -e PLAYWRIGHT_BROWSERS_PATH=/ms-playwright)

cleanup() {
  if [ "${KEEP:-0}" != "1" ]; then
    docker rm -f "$WEB" "$PG" >/dev/null 2>&1 || true
    docker network rm "$NET" >/dev/null 2>&1 || true
  fi
  # the volume mount can leave an empty node_modules mount-point on the host bind
  rmdir design-system/verify/harness/node_modules 2>/dev/null || true
}
trap cleanup EXIT

echo "== 1. render-goldens -> static HTML =="
docker run --rm -v "$REPO":/src -w /src "${GO_CACHE[@]}" "$GO_IMAGE" \
  sh -c "go run ./design-system/verify/harness/render-goldens -out design-system/goldens/inventory.html"

echo "== 1b. npm deps (pixelmatch/pngjs/playwright) in pinned image =="
docker run --rm "${HARNESS_MNT[@]}" -e PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 "$PW_IMAGE" \
  sh -c "npm install --no-audit --no-fund >/dev/null 2>&1 && echo deps ok"

if [ "${GOLDENS:-}" = "write" ]; then
  echo "== 2. capture --write-goldens (file://) =="
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --page /src/design-system/goldens/inventory.html
fi

echo "== 3a. Postgres (pinned) =="
docker network create "$NET" >/dev/null 2>&1 || true
docker rm -f "$PG" >/dev/null 2>&1 || true
docker run -d --name "$PG" --network "$NET" \
  -e POSTGRES_PASSWORD=verge -e POSTGRES_DB=verge -e POSTGRES_USER=verge "$PG_IMAGE" >/dev/null
for i in $(seq 1 60); do docker exec "$PG" pg_isready -U verge >/dev/null 2>&1 && break; sleep 1; done

echo "== 3b. build web binary =="
docker run --rm -v "$REPO":/src -w /src "${GO_CACHE[@]}" -v "${BIN_VOL}:/out" "$GO_IMAGE" \
  sh -c "go build -o /out/web ./cmd/web"

echo "== 3c. migrate + seed fixtures (dev operator) =="
docker run --rm --network "$NET" -v "$REPO":/src -w /src -v "${BIN_VOL}:/out" \
  -e VERGE_DEV=1 -e "DATABASE_URL=${DBURL}" -e VERGE_STATE_DIR=/tmp/verge-state "$GO_IMAGE" \
  /out/web -seed-fixtures design-system/fixtures/fixtures.json

echo "== 3d. serve web =="
docker rm -f "$WEB" >/dev/null 2>&1 || true
docker run -d --name "$WEB" --network "$NET" -v "$REPO":/src -w /src -v "${BIN_VOL}:/out" \
  -e VERGE_DEV=1 -e "DATABASE_URL=${DBURL}" -e VERGE_STATE_DIR=/tmp/verge-state "$GO_IMAGE" /out/web >/dev/null
for i in $(seq 1 30); do docker exec "$WEB" /out/web -healthcheck >/dev/null 2>&1 && break; sleep 1; done

# ADVISORY=1 (default) prints diffs and always exits 0 — the local Windows/macOS
# posture, where pixel output is not the canonical verdict. CI sets ADVISORY=0 so
# capture exits non-zero when any state-theme is over config.json's threshold — the
# binding G2 gate, which only ever runs in this pinned container.
ADV_FLAG=""
if [ "${ADVISORY:-1}" = "1" ]; then ADV_FLAG="--advisory"; fi
echo "== 4. capture --mode candidate ${ADV_FLAG} =="
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --base "http://${WEB}:8080"

echo "== done (ADVISORY=${ADVISORY:-1}) =="
