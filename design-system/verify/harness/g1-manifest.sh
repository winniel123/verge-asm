#!/usr/bin/env bash
# g1-manifest.sh — the G1 (verbatim) gate for the design-owned view layer.
#
# G1 asserts that the design-owned artifacts the app serves — the frozen templates,
# tokens, fixtures, verify configs, and the materialized goldens — are byte-identical
# to what landed, so a local edit to a frozen file is caught in CI (WORKFLOW.md v4).
# It hashes those artifacts via `git ls-files -s`, i.e. their committed git BLOB object
# ids, NOT the working-tree bytes: blob ids are line-ending-normalized and identical on
# Windows and the Linux CI runner, so a CRLF checkout never trips a false mismatch.
#
# Repo-owned harness code (this dir), designfs.go, render-goldens' intermediate
# inventory.html, and the freely-edited prose docs are NOT design-owned and are excluded.
#
#   g1-manifest.sh --write   # (re)generate G1SUMS.txt — only when a new package lands
#   g1-manifest.sh --check   # CI: recompute and diff against G1SUMS.txt (exit 1 on drift)
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$REPO"
MANIFEST="design-system/verify/harness/G1SUMS.txt"

# The design-owned artifacts G1 freezes. Add a screen's goldens dir here as it lands.
PATHS=(
  design-system/templates
  design-system/tokens
  design-system/fixtures
  design-system/verify/config.json
  design-system/verify/states.json
  design-system/verify/PILOT-P4.0.md
  design-system/goldens/inventory
  design-system/goldens/error
  design-system/goldens/profile
  design-system/goldens/signin
  design-system/goldens/setup
  design-system/goldens/coverage
  design-system/goldens/exposure
  design-system/goldens/drift
  design-system/goldens/rundetail
  design-system/goldens/scope
  design-system/goldens/signals
  design-system/goldens/dashboard
  design-system/goldens/asset
  design-system/goldens/subjectdetail
  design-system/goldens/graph
  design-system/goldens/reports
  design-system/goldens/reportartifact
  design-system/goldens/inbox
  design-system/goldens/search
  design-system/goldens/onboarding
  design-system/goldens/firstrun
  design-system/goldens/settings
)

# "<blob-sha>  <path>", sorted by path, C locale for a stable order across machines.
gen() { git ls-files -s -- "${PATHS[@]}" | awk '{print $2"  "$4}' | LC_ALL=C sort -k2; }

case "${1:-}" in
  --write)
    gen > "$MANIFEST"
    echo "G1: wrote $MANIFEST ($(wc -l < "$MANIFEST" | tr -d ' ') design-owned artifacts)"
    ;;
  --check)
    tmp="$(mktemp)"
    gen > "$tmp"
    if diff -u "$MANIFEST" "$tmp"; then
      echo "G1 OK — design-owned artifacts are byte-identical to the manifest."
    else
      echo "G1 FAIL — a design-owned artifact drifted from $MANIFEST (diff above)." >&2
      echo "A frozen artifact changed. If a NEW design package landed, run --write; otherwise revert the edit (design owns these; see WORKFLOW.md)." >&2
      rm -f "$tmp"
      exit 1
    fi
    rm -f "$tmp"
    ;;
  *)
    echo "usage: $0 --write|--check" >&2
    exit 2
    ;;
esac
