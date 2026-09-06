#!/bin/sh
# One pin, not two: `guard` and `release` both run git-cliff, and a version that
# drifted between them would gate one body and publish another (ADR-0138, #1257).
set -eu

GIT_CLIFF_VERSION="2.14.1"
GIT_CLIFF_SHA256="cba6ae86f0a4205784eed8ef049fe53c904138806e33a1bda2e25026b17198eb"

usage() {
	cat <<'EOF'
usage: scripts/install-git-cliff.sh [<bindir>]

Downloads the pinned git-cliff release archive, checks it against the pinned
SHA-256, and installs the binary into <bindir> (default: $RUNNER_TEMP/bin).

Prints the install directory on stdout. In GitHub Actions the caller appends
that line to $GITHUB_PATH.
EOF
}

case "${1-}" in
-h | --help)
	usage
	exit 0
	;;
esac

bindir="${1-${RUNNER_TEMP-/tmp}/bin}"
archive="git-cliff-${GIT_CLIFF_VERSION}-x86_64-unknown-linux-musl.tar.gz"
workdir="$(mktemp -d)"

cd "$workdir"
curl --fail --silent --show-error --location \
	--output "$archive" \
	"https://github.com/orhun/git-cliff/releases/download/v${GIT_CLIFF_VERSION}/${archive}"
printf '%s  %s\n' "$GIT_CLIFF_SHA256" "$archive" | sha256sum --check --strict -
tar -xzf "$archive" "git-cliff-${GIT_CLIFF_VERSION}/git-cliff"

mkdir -p "$bindir"
install -m 0755 "git-cliff-${GIT_CLIFF_VERSION}/git-cliff" "$bindir/git-cliff"

cd /
rm -rf "$workdir"

printf '%s\n' "$bindir"
