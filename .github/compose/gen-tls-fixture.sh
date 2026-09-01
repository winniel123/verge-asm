#!/usr/bin/env bash
# Generate the throwaway TLS chain the live-TLS boot job serves from `verge-db`
# (packaging-and-configuration.md §9.6).
#
#   bash .github/compose/gen-tls-fixture.sh <output-directory> [label]
#
# It writes root.crt, root.key, server.crt and server.key into <output-directory>.
# Nothing here ships to an operator, and no root lands in the repo tree: the job
# writes the chain under $RUNNER_TEMP and throws it away with the runner.
#
# The chain has two levels because ADR-0132 ruled the mounted object is a CA
# ROOT. A single self-signed server certificate used as its own sslrootcert
# would mount a leaf wearing a root's name, so the fixture would contradict the
# ADR it exists to prove.
#
# The server certificate carries SAN DNS:verge-db, and the DSN names that host.
# `verify-full` matches the certificate against the host named in the DSN, so a
# missing or mismatched SAN fails the handshake (§9.5).
#
# `label` only names the root's subject. Two runs always produce two unrelated
# roots, because each run mints a fresh key; the label is there so the wrong-CA
# negative run (#997) reads clearly in a log.
#
# The caller owns the key's OWNERSHIP. This sets mode 0600, because Postgres
# refuses a group- or world-readable key. The uid that must own it belongs to
# the server image rather than to the chain, so the job chowns it.
set -euo pipefail

out=${1:?usage: gen-tls-fixture.sh <output-directory> [label]}
label=${2:-ci}
host=verge-db

mkdir -p "$out"

# The CA root. It signs the server certificate below, and it is the only file
# docker-compose.external-db.yml mounts into web and worker.
openssl req -x509 -newkey rsa:2048 -sha256 -nodes -days 1 \
  -keyout "$out/root.key" -out "$out/root.crt" \
  -subj "/CN=verge-asm $label root" \
  -addext "basicConstraints=critical,CA:TRUE,pathlen:0" \
  -addext "keyUsage=critical,keyCertSign,cRLSign"

# The server certificate the root signs. Postgres serves this one.
openssl req -newkey rsa:2048 -sha256 -nodes \
  -keyout "$out/server.key" -out "$out/server.csr" \
  -subj "/CN=$host"

cat >"$out/server.ext" <<EXT
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:$host
EXT

openssl x509 -req -in "$out/server.csr" -sha256 -days 1 \
  -CA "$out/root.crt" -CAkey "$out/root.key" -CAcreateserial \
  -extfile "$out/server.ext" -out "$out/server.crt"

chmod 600 "$out/root.key" "$out/server.key"
chmod 644 "$out/root.crt" "$out/server.crt"

# Fail here rather than three steps later. A chain that does not verify, or a
# missing SAN, otherwise surfaces as a TLS handshake error inside a container,
# where the cause is much harder to read.
openssl verify -CAfile "$out/root.crt" "$out/server.crt"
san=$(openssl x509 -in "$out/server.crt" -noout -ext subjectAltName)
case "$san" in
  *"DNS:$host"*) ;;
  *) echo "gen-tls-fixture: the server certificate carries no SAN DNS:$host" >&2; exit 1 ;;
esac

echo "gen-tls-fixture: $out holds the chain \"verge-asm $label root\" -> $host"
