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
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen subjectdetail -outdir design-system/goldens/subjectdetail && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen reports -outdir design-system/goldens/reports && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen reportartifact -outdir design-system/goldens/reportartifact && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen inbox -outdir design-system/goldens/inbox && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen onboarding -outdir design-system/goldens/onboarding && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen firstrun -outdir design-system/goldens/firstrun && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen search -outdir design-system/goldens/search && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen graph -out design-system/goldens/graph.html && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen settings -outdir design-system/goldens/settings && \
         go run -buildvcs=false ./design-system/verify/harness/render-goldens -screen shell -outdir design-system/goldens/shell"

echo "== 1b. npm deps (pixelmatch/pngjs/playwright) in pinned image =="
docker run --rm "${HARNESS_MNT[@]}" -e PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 "$PW_IMAGE" \
  sh -c "npm install --no-audit --no-fund >/dev/null 2>&1 && echo deps ok"

if [ "${GOLDENS:-}" = "write" ]; then
  echo "== 2. capture --write-goldens (file://) — inventory + error + profile + signin + setup + coverage + exposure + drift + rundetail =="
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen inventory --page /src/design-system/goldens/inventory.html
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen error --pagedir /src/design-system/goldens/error
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen profile --pagedir /src/design-system/goldens/profile
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen signin --pagedir /src/design-system/goldens/signin
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen setup --pagedir /src/design-system/goldens/setup
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen coverage --pagedir /src/design-system/goldens/coverage
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen exposure --pagedir /src/design-system/goldens/exposure
  # drift is a SINGLE shared golden file (--page, like inventory): its default/feed-expanded/
  # range-open states are the same HTML with the frozen tmpl's own JS (group-collapse, range
  # popover) driven over it by capture.mjs (states.json). So write one page and let the state
  # JS produce the expanded/open captures — not a per-state dir.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen drift --page /src/design-system/goldens/drift.html
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen rundetail --pagedir /src/design-system/goldens/rundetail
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen scope --pagedir /src/design-system/goldens/scope
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen signals --pagedir /src/design-system/goldens/signals
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen dashboard --pagedir /src/design-system/goldens/dashboard
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen asset --pagedir /src/design-system/goldens/asset
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen subjectdetail --pagedir /src/design-system/goldens/subjectdetail
  # reports: per-state HTML dir (--pagedir). default/range-open/row-menu-open share the "reports"
  # page HTML (their JS is driven on both sides); wizard-1..4 are the per-step "schedulewizard"
  # renders at the PRG GET URLs (states.json).
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen reports --pagedir /src/design-system/goldens/reports
  # reportartifact: per-state HTML dir (--pagedir). default is the delivered document; never-delivered
  # is the empty-state document (schedule s2, .Doc.Empty) — both static server renders of "reportartifact".
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen reportartifact --pagedir /src/design-system/goldens/reportartifact
  # inbox: per-state HTML dir (--pagedir). default/message-open/unread-filter are three static server
  # renders of "inbox" (no .Body per SPEC-CHANGE #24 — the detail is census + delivery receipts); the
  # ?id / ?filter selection is baked into each render, so no state JS runs.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen inbox --pagedir /src/design-system/goldens/inbox
  # onboarding: per-state HTML dir (--pagedir). wizard-1..4 are the per-step "onboarding" renders at
  # the PRG GET URLs (states.json); each is a static server render of the self-contained tmpl.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen onboarding --pagedir /src/design-system/goldens/onboarding
  # firstrun: per-state HTML dir (--pagedir). default is the empty-estate wrap of `/` (dashboard.tmpl
  # "home" wrapping the bare "firstrun" define when .EmptyEstate) — one static server render.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen firstrun --pagedir /src/design-system/goldens/firstrun
  # graph is a SINGLE shared golden file (--page, like drift): its default / node-drawer /
  # filtered-critical states are the same HTML with the frozen tmpl's own view JS (node click →
  # drawer, severity listbox → filter) driven over it by capture.mjs (states.json) on BOTH sides.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen graph --page /src/design-system/goldens/graph.html
  # search: per-state HTML dir (--pagedir). default (/search?q=acme) / empty (/search?q=zzz-none) are two
  # static server renders of "search" — the ?q= query is baked into each render (segments folded through
  # the #25a builder), so no state JS runs.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen search --pagedir /src/design-system/goldens/search
  # settings: per-state HTML dir (--pagedir). The 18 chrome-hosted sub-tab/dialog states are static
  # server renders of "settings" (the ?tab=/dialog selection baked into each render); the 19th
  # (forbidden) is the error-page settings-forbidden. The dialog/drawer states crop `body`, the rest
  # crop `main` (states.json) — no state JS runs.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen settings --pagedir /src/design-system/goldens/settings
  # shell (#27f): the 6 captured shell-state goldens on `/` FULL-PAGE (default / palette-open /
  # bell-open / acct-open / scan-running / toasts). The org switcher is retired (SPEC-CHANGE #33,
  # v3.17.0): shell.tmpl renders only the static org chip, so there is no org-open state.
  docker run --rm "${HARNESS_MNT[@]}" "$PW_IMAGE" \
    node capture.mjs --full-page --mode golden --write-goldens --advisory --screen shell --pagedir /src/design-system/goldens/shell
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
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen inventory --base "http://${WEB}:8080"
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen error --base "http://${WEB}:8080"
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen profile --base "http://${WEB}:8080" --adopt /dev/profile/session
# signin: chrome-less auth surfaces (crop=body); per-state session none/viewer via the /dev/session
# mint, no --adopt/--hide-chrome. The no-sso variant rides a dev ?variant query capture.mjs appends.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen signin --base "http://${WEB}:8080"
# coverage: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# DB-backed session. drops the sticky console header from flow so <main> sits at the
# viewport top and aligns with the chrome-less golden (as profile). MUST come BEFORE setup: the
# empty state hits /dev/seed/empty-authed (keeps accounts), but setup's candidate then hits
# /dev/seed/empty which TRUNCATEs the account table — capturing coverage after that would strand it
# with no authed-admin session.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen coverage --base "http://${WEB}:8080"
# exposure: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json exposure slice under VERGE_DEV. drops the
# sticky console header from flow so <main> sits at the viewport top and aligns with the
# chrome-less golden (as coverage). The withheld state rides a dev ?variant=no-internet-vantage
# query capture.mjs appends (states.json), which exposurePage reads to render WITHHELD. No seed —
# it touches no table — so its position relative to coverage is free; it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand exposure's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen exposure --base "http://${WEB}:8080"
# drift: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json drift slice under VERGE_DEV. drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less
# golden (as coverage). The feed-expanded / range-open states run states.json's `js` (expand the
# collapsed group headers / open the range popover) against the frozen tmpl's own handlers, in BOTH
# golden and candidate. No seed — it touches no table — so its position relative to coverage is
# free; it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand drift's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen drift --base "http://${WEB}:8080"
# rundetail: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# in-memory dev fixture (no DB reshape). drops the sticky console header from flow so
# <main> sits at the viewport top and aligns with the chrome-less golden (as coverage). Routes are
# /runs/1407 (default, complete) and /runs/1409 (running, #35); 1408 is the MISSING id the error
# screen already covers. MUST come BEFORE setup: setup's candidate hits /dev/seed/empty which
# TRUNCATEs accounts, stranding the admin session
# the /dev/session/admin mint needs.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen rundetail --base "http://${WEB}:8080"
# scope: chrome-hosted screen cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json scope slice under VERGE_DEV. drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less
# golden (as coverage). The refusal / exclusion-preview states run states.json's `js` (post the /20
# through the seed form; type staging-4 + click Preview) against the frozen tmpl's own forms, which
# declareSeed / previewExclusion answer in devMode with the pinned fixture + overlay. No seed — it
# touches no table — so its position relative to coverage is free; it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand scope's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen scope --base "http://${WEB}:8080"
# signals: chrome-hosted screen, session admin (per-state /dev/session mint), served from the pinned
# fixtures.json signals slice under VERGE_DEV. drops the sticky console header from flow
# so <main> sits at the viewport top and aligns with the chrome-less golden (as scope). The default /
# withdrawn-tab / menu-open states crop `main`; the drawer-open / drawer-annotated / descope-confirm
# states crop `body` (per-state crop in states.json) because the fixed scrim + drawer / dialog escape
# <main>. The drawer / descope / withdrawn states are pure query-string routes signalsFixtureData
# reads; menu-open runs states.json's `js` (open a kebab) against the frozen tmpl's own handler on
# BOTH sides. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty
# TRUNCATEs the account table (which would strand signals' authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen signals --base "http://${WEB}:8080"
# dashboard: chrome-hosted `/` cropped to `main`, session admin (per-state /dev/session mint), served
# from the pinned fixtures.json dashboard slice under VERGE_DEV. drops the sticky console
# header from flow so <main> sits at the viewport top and aligns with the chrome-less golden (as
# coverage). The scanning state rides a dev ?variant=scanning query capture.mjs appends (states.json),
# which home() reads to light .Scanning + .ScanDetail; banner-dismissed is the pure ?probe=dismissed
# route. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the
# account table (which would strand dashboard's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen dashboard --base "http://${WEB}:8080"
# asset: chrome-hosted /asset/{key} cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json asset slice under VERGE_DEV. drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less golden
# (as coverage). The default state is the pure /asset/edge-gw-03.acmecorp.io route assetPage reads in
# devMode. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs
# the account table (which would strand asset's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen asset --base "http://${WEB}:8080"
# subjectdetail: chrome-hosted /subjects/{service,endpoint} cropped to `main`, session admin
# (per-state /dev/session mint), served from the pinned fixtures.json subjectdetail slices under
# VERGE_DEV. drops the sticky console header from flow so <main> sits at the viewport
# top and aligns with the chrome-less golden (as asset). The service / endpoint / service-withdrawn
# states are pure ?key= routes servicePage/endpointPage read in devMode. No seed — it touches no
# table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand subjectdetail's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen subjectdetail --base "http://${WEB}:8080"
# graph: chrome-hosted /graph, session admin (per-state /dev/session mint), served from the pinned
# fixtures.json graph slice under VERGE_DEV. drops the sticky console header so <main>
# sits at the viewport top and aligns with the chrome-less golden (as coverage). The default /
# filtered-critical states crop `main`; the node-drawer state crops `body` (per-state crop in
# states.json) because the fixed scrim + drawer escape <main>. All three run states.json's `js` (click
# a node / pick the critical severity option) against the frozen tmpl's own view JS on BOTH golden and
# candidate. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs
# the account table (which would strand graph's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen graph --base "http://${WEB}:8080"
# reports: chrome-hosted /reports (+ the schedule wizard) cropped to `main`, session admin (per-state
# /dev/session mint), served from the pinned fixtures.json reports slice under VERGE_DEV.
# drops the sticky console header so <main> sits at the viewport top and aligns with the chrome-less
# golden (as coverage). default/range-open/row-menu-open are the /reports route driven by the frozen
# tmpl's own JS (open the range popover / open a row kebab) on BOTH sides; wizard-1..4 are the pure PRG
# GET URLs (/reports/schedule/new?step=N&…) newReportScheduleWizard reads in devMode. No seed — it
# touches no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table
# (which would strand reports' authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen reports --base "http://${WEB}:8080"
# reportartifact: chrome-hosted /reports/delivery cropped to `main`, session admin (per-state
# /dev/session mint), served from the pinned fixtures.json reportartifact slice under VERGE_DEV.
# drops the sticky console header so <main> sits at the viewport top and aligns with the
# chrome-less golden (as reports). default is the pure /reports/delivery route; never-delivered rides a
# dev ?variant=never-delivered query capture.mjs appends (states.json), which reportDeliveryPage reads
# to serve the .Doc.Empty document for schedule s2. No seed — it touches no table — so it MUST precede
# setup, whose /dev/seed/empty TRUNCATEs the account table (which would strand its authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen reportartifact --base "http://${WEB}:8080"
# inbox: chrome-hosted /inbox cropped to `main`, session admin (per-state /dev/session mint), served
# from the pinned fixtures.json inbox slice under VERGE_DEV. drops the sticky console
# header so <main> sits at the viewport top and aligns with the chrome-less golden (as reports).
# default is the plain /inbox route; message-open is the pure /inbox?id=m1 route and unread-filter the
# pure /inbox?filter=unread route inboxFixtureData reads in devMode (no state JS — the ?id/?filter
# selection is server-rendered). No seed — it touches no table — so it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand inbox's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen inbox --base "http://${WEB}:8080"
# onboarding: chrome-hosted /onboarding cropped to `main`, session admin (per-state /dev/session
# mint). drops the sticky console header so <main> sits at the viewport top and aligns
# with the chrome-less golden (as reports). wizard-1..4 are the pure PRG GET URLs (/onboarding?step=N&…)
# the onboarding GET handler reconstructs from the query — a stateless read (no DB reshape), so its
# constants (cadence presets, default cad, profile) already match the fixture. No seed — it touches no
# table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand onboarding's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen onboarding --base "http://${WEB}:8080"
# firstrun: chrome-hosted `/` cropped to `main`, session admin (per-state /dev/session mint), served
# from the pinned fixtures.json firstrun slice under VERGE_DEV. drops the sticky console
# header so <main> sits at the viewport top and aligns with the chrome-less golden (as reports). The
# default state rides a dev ?variant=empty-estate query capture.mjs appends (states.json), which home()
# reads to serve the empty-estate wrap (dashboard.tmpl "home" wrapping "firstrun"). No seed — it touches
# no table — so it MUST precede setup, whose /dev/seed/empty TRUNCATEs the account table (which would
# strand firstrun's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen firstrun --base "http://${WEB}:8080"
# search: chrome-hosted /search cropped to `main`, session admin (per-state /dev/session mint), served
# from the pinned fixtures.json search slice under VERGE_DEV. drops the sticky console
# header from flow so <main> sits at the viewport top and aligns with the chrome-less golden (as inbox).
# default (/search?q=acme) and empty (/search?q=zzz-none) are pure ?q= routes searchFixtureData reads in
# devMode — no seed, no state JS. No table touched — so it MUST precede setup, whose /dev/seed/empty
# TRUNCATEs the account table (which would strand search's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen search --base "http://${WEB}:8080"
# settings: chrome-hosted /settings cropped to `main`, session admin (per-state /dev/session mint),
# served from the pinned fixtures.json settings slice under VERGE_DEV. drops the sticky
# console header from flow so <main> sits at the viewport top and aligns with the chrome-less golden
# (as reports). The 18 sub-tab/dialog states are pure ?tab=/dialog routes settingsFixtureData reads in
# devMode (the dialog/drawer states — team-invite/team-remove/sessions-revoke-all/sources-consent/
# integrations-drawer — crop `body`; the rest crop `main`); the 19th (forbidden) is the viewer session
# at /settings?tab=team, which requireSettingsAdmin refuses with the error-page settings-forbidden. No
# state JS runs. No seed — it touches no table — so it MUST precede setup, whose /dev/seed/empty
# TRUNCATEs the account table (which would strand settings' authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen settings --base "http://${WEB}:8080"
# shell (#27f): the design-owned chrome itself, FULL-PAGE on `/` (dashboard fixture), session admin
# (per-state /dev/session mint). The 6 states drive the shell's own JS on the live candidate: default
# is the plain render; palette-open/bell-open/acct-open run states.json's `js` (open the ⌘K palette /
# bell popover / account menu) against the frozen tmpl's own handlers; scan-running rides
# ?variant=scanning (home lights .Scanning → chrome .ScanRunning); toasts rides ?variant=flash-toast
# (injectChrome folds the fixture toast stack into .Chrome.Toasts) with a 400ms delay to capture it
# before the 5s auto-dismiss. The org switcher is retired (static chip only, SPEC-CHANGE #33), so
# there is no org-open state. No seed — it touches no table — so it MUST precede setup, whose
# /dev/seed/empty TRUNCATEs the account table (which would strand the shell's authed-admin session).
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen shell --base "http://${WEB}:8080"
# setup: chrome-less first-run surface (crop=body). states.json setup declares seed:"empty" —
# capture.mjs hits /dev/seed/empty (VERGE_DEV) to empty the account table and reopen the setup
# window before each state. MUST be the LAST candidate capture: emptying the shared fixture DB
# here would strand any screen captured after it.
docker run --rm --network "$NET" "${HARNESS_MNT[@]}" "$PW_IMAGE" \
  node capture.mjs --full-page --mode candidate $ADV_FLAG --screen setup --base "http://${WEB}:8080"

echo "== done (ADVISORY=${ADVISORY:-1}) =="
