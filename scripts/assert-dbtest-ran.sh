#!/usr/bin/env bash
# Fail when the internal/dbtest tier reached no database. A skipped case exits 0
# exactly as a passing one does, so without this a lost DSN turns the tier into a
# green no-op wherever it runs (#2255).
#
# Shared by the required `test` job and scripts/dbtest.sh, so the CI guard and the
# local one cannot drift apart.
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: ${0##*/} <go-test-verbose-log>" >&2
  exit 2
fi

log="$1"

if [ ! -f "$log" ]; then
  echo "::error::$log does not exist, so the harness produced no log to judge"
  exit 1
fi

if grep -q -- "--- SKIP" "$log"; then
  echo "::error::the harness skipped, so it reached no database"
  grep -- "--- SKIP" "$log" >&2 || true
  exit 1
fi

# A package with no runnable case prints no SKIP either, so the guard asserts the
# positive: at least one case reached the database.
passed=$(grep -c -- "--- PASS" "$log" || true)
if [ "$passed" -eq 0 ]; then
  echo "::error::the harness ran no case, so it reached no database"
  exit 1
fi

echo "the harness ran $passed case(s) against a database"
