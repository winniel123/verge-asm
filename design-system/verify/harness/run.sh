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

echo "== 1. render-goldens -> static HTML (inventory + error + profile + signin + setup + coverage + exposure + drift + rundetail) =="
docker run --rm -v "$REPO":/src -w /src "${GO_CACHE[@]}" "$GO_IMAGE" \
  sh -c "go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen inventory -out design-system/goldens/inventory.html && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen error -outdir design-system/goldens/error && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen profile -outdir design-system/goldens/profile && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen signin -outdir design-system/goldens/signin && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen setup -outdir design-system/goldens/setup && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen coverage -outdir design-system/goldens/coverage && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen exposure -outdir design-system/goldens/exposure && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen drift -out design-system/goldens/drift.html && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen rundetail -outdir design-system/goldens/rundetail && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen scope -outdir design-system/goldens/scope && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen signals -outdir design-system/goldens/signals && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen dashboard -outdir design-system/goldens/dashboard && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen asset -outdir design-system/goldens/asset && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen subjectdetail -outdir design-system/goldens/subjectdetail"

echo "== 1b. npm deps (pixelmatch/pngjs/playwright) in pinned image =="
docker run --rm "${HARNESS_MNT[@]}" -e PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 "$PW_IMAGE" \
  sh -c "npm install --no-audit --no-fund >/dev/null 2>&1 && echo deps ok"

if [ "${GOLDENS:-}" = "write" ]; then
  echo "== 2. capture --write-goldens (file://) — inventory + error + profile + signin + setup + coverage + exposure + drift + rundetail =="
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen inventory --page /src/design-system/goldens/inventory.html
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen error --pagedir /src/design-system/goldens/error
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen profile --pagedir /src/design-system/goldens/profile
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen signin --pagedir /src/design-system/goldens/signin
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen setup --pagedir /src/design-system/goldens/setup
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen coverage --pagedir /src/design-system/goldens/coverage
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen exposure --pagedir /src/design-system/goldens/exposure
  # drift is a SINGLE shared golden file (--page, like inventory): its default/feed-expanded/
  # range-open states are the same HTML with the frozen tmpl's own JS (group-collapse, range
  # popover) driven over it by capture.mjs (states.json). So write one page and let the state
  # JS produce the expanded/open captures — not a per-state dir.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen drift --page /src/design-system/goldens/drift.html
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen rundetail --pagedir /src/design-system/goldens/rundetail
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen scope --pagedir /src/design-system/goldens/scope
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen signals --pagedir /src/design-system/goldens/signals
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen dashboard --pagedir /src/design-system/goldens/dashboard
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen asset --pagedir /src/design-system/goldens/asset
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --mode golden --write-goldens --advisory --screen subjectdetail --pagedir /src/design-system/goldens/subjectdetail
fi

echo "== 3a. Postgres (pinned) =="
docker network create "$NET" >/dev/null 2>&1 || true
docker rm -f "$PG" >/dev/null 2>&1 || true
docker run -d --name "$PG" --network "$NET" \
  -e POSTGRES_PASSWORD=verge -e POSTGRES_DB=verge -e POSTGRES_USER=verge "$PG_IMAGE" >/dev/null
for i in $(seq 1 60); do docker exec "$PG" pg_isready -U verge >/dev/null 2>&1 && break; sleep 1; done

echo "== 3b. build web binary =="
docker run --rm -v "$REPO":/src -w /src "${GO_CACHE[@]}" -v "${BIN_VOL}:/out" "$GO_IMAGE" \
  sh -c "go build -buildvcs=false -o /out/web ./cmd/web"

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
echo "== 4. capture --mode candidate ${ADV_FLAG} — inventory + error + profile + signin + coverage + exposure + drift + rundetail + setup =="
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen inventory --base "http://${WEB}:8080"
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen error --base "http://${WEB}:8080"
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen profile --base "http://${WEB}:8080" --adopt /dev/profile/session --hide-chrome
# signin: chrome-less auth surfaces (crop=body); per-state session none/viewer via the /dev/session
# mint, no --adopt/--hide-chrome. The no-sso variant rides a dev ?variant query capture.mjs appends.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen signin --base "http://${WEB}:8080"
# coverage: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# DB-backed session. --hide-chrome drops the sticky console header from flow so <main> sits at the
# viewport top and aligns with the chrome-less golden (as profile). MUST come BEFORE setup: the
# empty state hits /dev/seed/empty-authed (keeps accounts), but setup's candidate then hits
# /dev/seed/empty which TRUNCATEs the account table — capturing coverage after that would strand it
# with no authed-admin session.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen coverage --base "http://${WEB}:8080" --hide-chrome
# exposure: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json exposure slice under VERGE_DEV. --hide-chrome drops the
# sticky console header from flow so <main> sits at the viewport top and aligns with the
# chrome-less golden (as coverage). The withheld state rides a dev ?variant=no-internet-vantage
# query capture.mjs appends (states.json), which exposurePage reads to render WITHHELD. No seed —
# it touches no table — so its position relative to coverage is free; it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand exposure's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen exposure --base "http://${WEB}:8080" --hide-chrome
# drift: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json drift slice under VERGE_DEV. --hide-chrome drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less
# golden (as coverage). The feed-expanded / range-open states run states.json's `js` (expand the
# collapsed group headers / open the range popover) against the frozen tmpl's own handlers, in BOTH
# golden and candidate. No seed — it touches no table — so its position relative to coverage is
# free; it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand drift's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen drift --base "http://${WEB}:8080" --hide-chrome
# rundetail: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# in-memory dev fixture (no DB reshape). --hide-chrome drops the sticky console header from flow so
# <main> sits at the viewport top and aligns with the chrome-less golden (as coverage). Route is
# /runs/1407 (states.json); 1408 is the MISSING id the error screen already covers. MUST come BEFORE
# setup: setup's candidate hits /dev/seed/empty which TRUNCATEs accounts, stranding the admin session
# the /dev/session/admin mint needs.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen rundetail --base "http://${WEB}:8080" --hide-chrome
# scope: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json scope slice under VERGE_DEV. --hide-chrome drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less
# golden (as coverage). The refusal / exclusion-preview states run states.json's `js` (post the /20
# through the seed form; type staging-4 + click Preview) against the frozen tmpl's own forms, which
# declareSeed / previewExclusion answer in devMode with the pinned fixture + overlay. No seed — it
# touches no table — so its position relative to coverage is free; it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand scope's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen scope --base "http://${WEB}:8080" --hide-chrome
# signals: chrome-hosted screen, session admin (per-state /dev/session mint), served from the pinned
# fixtures.json signals slice under VERGE_DEV. --hide-chrome drops the sticky console header from flow
# so <main> sits at the viewport top and aligns with the chrome-less golden (as scope). The default /
# withdrawn-tab / menu-open states crop `main`; the drawer-open / drawer-annotated / descope-confirm
# states crop `body` (per-state crop in states.json) because the fixed scrim + drawer / dialog escape
# <main>. The drawer / descope / withdrawn states are pure query-string routes signalsFixtureData
# reads; menu-open runs states.json's `js` (open a kebab) against the frozen tmpl's own handler on
# BOTH sides. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty
# TRUNCATEs the account table (which would strand signals' authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen signals --base "http://${WEB}:8080" --hide-chrome
# dashboard: chrome-hosted `/` cropped to `main`, session admin (per-state /dev/session mint), served
# from the pinned fixtures.json dashboard slice under VERGE_DEV. --hide-chrome drops the sticky console
# header from flow so <main> sits at the viewport top and aligns with the chrome-less golden (as
# coverage). The scanning state rides a dev ?variant=scanning query capture.mjs appends (states.json),
# which home() reads to light .Scanning + .ScanDetail; banner-dismissed is the pure ?probe=dismissed
# route. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the
# account table (which would strand dashboard's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen dashboard --base "http://${WEB}:8080" --hide-chrome
# asset: chrome-hosted /asset/{key} cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json asset slice under VERGE_DEV. --hide-chrome drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less golden
# (as coverage). The default state is the pure /asset/edge-gw-03.acmecorp.io route assetPage reads in
# devMode. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs
# the account table (which would strand asset's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen asset --base "http://${WEB}:8080" --hide-chrome
# subjectdetail: chrome-hosted /subjects/{service,endpoint} cropped to `main`, session admin
# (per-state /dev/session mint), served from the pinned fixtures.json subjectdetail slices under
# VERGE_DEV. --hide-chrome drops the sticky console header from flow so <main> sits at the viewport
# top and aligns with the chrome-less golden (as asset). The service / endpoint / service-withdrawn
# states are pure ?key= routes servicePage/endpointPage read in devMode. No seed — it touches no
# table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand subjectdetail's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen subjectdetail --base "http://${WEB}:8080" --hide-chrome
# setup: chrome-less first-run surface (crop=body). states.json setup declares seed:"empty" —
# capture.mjs hits /dev/seed/empty (VERGE_DEV) to empty the account table and reopen the setup
# window before each state. MUST be the LAST candidate capture: emptying the shared fixture DB
# here would strand any screen captured after it.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --mode candidate $ADV_FLAG --screen setup --base "http://${WEB}:8080"

echo "== done (ADVISORY=${ADVISORY:-1}) =="
