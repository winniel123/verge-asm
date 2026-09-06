#!/bin/sh
# POSIX sh, not Go: `commentlint` runs as `go run` after `setup-go`, so a Go
# checker would be built by the toolchain it judges (ADR-0001, #1247).
set -eu

usage() {
	cat <<'EOF'
usage: scripts/check-go-pins.sh [--mode ci|release]

Checks the Go toolchain pins against .go-version:

  a  the builder digest's org.opencontainers.image.version annotation == .go-version   (the gate)
  b  the builder FROM's decorative tag                              == .go-version   (hygiene)
  c  go.mod's `go` line                                             <= .go-version   (the floor)

Compare (a) needs one manifest fetch from Docker Hub. --mode ci retries and then
skips it with a warning; --mode release retries and then fails. (b) and (c) block
in both modes. The mode also reads from GO_PIN_MODE; the flag wins.
EOF
}

mode="${GO_PIN_MODE:-ci}"
while [ "$#" -gt 0 ]; do
	case "$1" in
	--mode)
		[ "$#" -ge 2 ] || {
			echo "check-go-pins: --mode needs a value" >&2
			exit 2
		}
		mode="$2"
		shift 2
		;;
	--mode=*)
		mode="${1#--mode=}"
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "check-go-pins: unknown argument: $1" >&2
		usage >&2
		exit 2
		;;
	esac
done

case "$mode" in
ci | release) ;;
*)
	echo "check-go-pins: mode must be ci or release, got: $mode" >&2
	exit 2
	;;
esac

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
go_version_file="$root/.go-version"
dockerfile="$root/Dockerfile"
gomod="$root/go.mod"

fail_count=0

report() {
	# $1 = level (error|warning), $2 = message
	if [ "${GITHUB_ACTIONS:-}" = "true" ]; then
		printf '::%s::%s\n' "$1" "$2"
	fi
	printf '%s: %s\n' "$1" "$2" >&2
}

fail() {
	report error "$1"
	fail_count=$((fail_count + 1))
}

die() {
	report error "$1"
	exit 1
}

# --- the source of truth ---------------------------------------------------

[ -f "$go_version_file" ] || die ".go-version is missing at $go_version_file"

# A bare scalar with no `go` prefix and no `v`: actions/setup-go passes the whole
# trimmed file through as the version spec, so any extra token becomes a bad spec.
go_version=$(tr -d ' \t\r' <"$go_version_file" | sed '/^$/d')
case $(printf '%s\n' "$go_version" | wc -l | tr -d ' ') in
1) ;;
*) die ".go-version must hold exactly one line, found: $(printf '%s' "$go_version" | tr '\n' ' ')" ;;
esac
case "$go_version" in
[0-9]*.[0-9]*.[0-9]*) ;;
*) die ".go-version must hold a bare MAJOR.MINOR.PATCH scalar, found: $go_version" ;;
esac

echo "check-go-pins: mode=$mode .go-version=$go_version"

# --- the builder reference -------------------------------------------------

[ -f "$dockerfile" ] || die "Dockerfile is missing at $dockerfile"

builder_ref=$(sed -n \
	's/^FROM .*[[:space:]]\(golang:[^[:space:]@]*@sha256:[0-9a-f]\{64\}\).*/\1/p' \
	"$dockerfile" | head -n 1)
[ -n "$builder_ref" ] || die "Dockerfile has no digest-pinned golang builder FROM"

builder_tag=${builder_ref#golang:}
builder_tag=${builder_tag%@*}
builder_digest=${builder_ref#*@}

# --- compare c: the go.mod floor -------------------------------------------

[ -f "$gomod" ] || die "go.mod is missing at $gomod"

gomod_go=$(sed -n 's/^go[[:space:]]\{1,\}\([0-9][0-9.]*\).*/\1/p' "$gomod" | head -n 1)
[ -n "$gomod_go" ] || die "go.mod has no \`go\` line"

# Sortable zero-padded key, so 1.26.10 orders above 1.26.8.
version_key() {
	printf '%s' "$1" | awk -F. '{printf "%05d%05d%05d\n", $1, ($2 == "" ? 0 : $2), ($3 == "" ? 0 : $3)}'
}

if [ "$(version_key "$gomod_go")" -le "$(version_key "$go_version")" ]; then
	echo "check-go-pins: c ok   go.mod floor go $gomod_go <= .go-version $go_version"
else
	fail "c: go.mod's floor go $gomod_go is above .go-version $go_version"
fi

# --- compare b: the decorative tag -----------------------------------------

builder_tag_version=${builder_tag%%-*}
if [ "$builder_tag_version" = "$go_version" ]; then
	echo "check-go-pins: b ok   Dockerfile builder tag golang:$builder_tag agrees with .go-version"
else
	fail "b: the Dockerfile builder tag golang:$builder_tag names $builder_tag_version, not $go_version"
fi

# --- compare a: the digest's own annotation --------------------------------

# A tag written beside a digest is decoration: Docker resolves by digest and never
# validates the tag, so only the annotation proves the builder (ADR-0001, #1247).
attempts=3
raw=$(mktemp)
fetch_err=$(mktemp)
trap 'rm -f "$raw" "$fetch_err"' EXIT INT TERM

fetch_manifest() {
	n=1
	while :; do
		if docker buildx imagetools inspect --raw "golang@$builder_digest" >"$raw" 2>"$fetch_err"; then
			return 0
		fi
		if [ "$n" -ge "$attempts" ]; then
			return 1
		fi
		sleep $((n * 5))
		n=$((n + 1))
	done
}

skip_a() {
	# GitHub-hosted runners share IPs that Docker Hub rate-limits, so a fetch
	# failure is advisory on a pull request and fatal on a release (ADR-0001).
	if [ "$mode" = "release" ]; then
		fail "a: $1, and a release must not publish on an unproven builder"
	else
		report warning "a: $1; compare (a) skipped, (b) and (c) still ran"
	fi
}

if ! command -v docker >/dev/null 2>&1; then
	skip_a "docker is not on PATH"
elif ! command -v jq >/dev/null 2>&1; then
	skip_a "jq is not on PATH"
elif ! fetch_manifest; then
	skip_a "the manifest fetch for golang@$builder_digest failed after $attempts attempts: $(tr '\n' ' ' <"$fetch_err")"
else
	# The per-platform manifests carry the annotation on an index. A single-platform
	# digest has no .manifests, so fall back to the top level rather than report a
	# missing annotation, which would block a release on a valid pin (#1247).
	annotated=$(jq -r '
		[ (.manifests[]?.annotations? // {}), (.annotations // {}) ]
		| map(.["org.opencontainers.image.version"] // empty)
		| unique | .[]
	' <"$raw")
	if [ -z "$annotated" ]; then
		# An absent annotation is a mismatch, not a fetch failure: the fetch
		# succeeded and the digest simply does not state a version.
		fail "a: golang@$builder_digest carries no org.opencontainers.image.version annotation"
	else
		mismatch=""
		for value in $annotated; do
			[ "${value%%-*}" = "$go_version" ] || mismatch="$mismatch $value"
		done
		if [ -n "$mismatch" ]; then
			fail "a: golang@$builder_digest is annotated$mismatch, not $go_version"
		else
			echo "check-go-pins: a ok   golang@$builder_digest is annotated $(printf '%s' "$annotated" | tr '\n' ' ')"
		fi
	fi
fi

if [ "$fail_count" -gt 0 ]; then
	echo "check-go-pins: $fail_count compare(s) failed" >&2
	exit 1
fi
echo "check-go-pins: pins agree"
