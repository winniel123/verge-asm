# Verifying verge-asm

How to build, test, regenerate code, and reproduce CI locally — i.e. how to *confirm*
a checkout is sound before you ship or open a PR.

The Go toolchain and `sqlc` are **not** required on your machine; everything runs
through Docker, matching the versions the images and CI pin. Run these from the repo
root. Examples use `"$PWD"` (Git Bash / macOS / Linux); in PowerShell use `${PWD}`.

The pipeline of record is [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).
Every job below mirrors one of its jobs.

---

## The fast loop: vet + test

```sh
docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm \
  sh -c 'go vet ./... && go test ./...'
```

This is CI's `test` job. `go test ./...` covers the whole tree, including the domain
packages under `internal/` and the handler tests under `cmd/web/`.

To iterate on one package, narrow the path:

```sh
docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm \
  go test ./internal/scan/...
```

> **Windows checkouts and byte-exact tests.** The golden/corpus tests compare exact
> bytes. If Git checked files out with CRLF line endings, those comparisons fail
> locally even though CI (which uses LF) is green. Fix the working tree with
> `git config core.autocrlf false` then `git rm --cached -r . && git reset --hard`,
> or simply run the tests inside the Linux container above, which sees the committed
> LF bytes regardless of how they appear on disk.

---

## Build every binary, every architecture

CI's `build` job compiles all three commands for both matrix architectures with the
exact pinned flags. Reproduce it:

```sh
for arch in amd64 arm64; do
  docker run --rm -v "$PWD":/src -w /src \
    -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=$arch -e GOAMD64=v1 -e GOARM64=v8.0 \
    golang:1.25-bookworm \
    sh -c 'go build -o /dev/null ./cmd/web &&
           go build -o /dev/null ./cmd/worker &&
           go build -o /dev/null ./cmd/prober'
done
```

The flags are not cosmetic: `CGO_ENABLED=0` fixes the resolver and keeps the pushed
prober statically linked, and `GOAMD64=v1` pins the floating-point behaviour so a
measurement value cannot move with the build host. See
[`docs/spec/packaging-and-configuration.md`](../spec/packaging-and-configuration.md) §1.

---

## Regenerate the sqlc code

`internal/db/*.sql.go` is **generated** from `db/queries/*.sql` against the migration
schema (`sqlc.yaml`). CI's `sqlc` job regenerates it and fails if the result differs
from what is committed — so after editing any query or migration, regenerate and
commit the diff:

```sh
docker run --rm -v "$PWD":/src -w /src sqlc/sqlc:1.31.1 generate
git diff -- internal/db          # review, then commit
```

To *check* (as CI does) without intending to change anything:

```sh
docker run --rm -v "$PWD":/src -w /src sqlc/sqlc:1.31.1 generate
git diff --exit-code -- internal/db   # non-zero exit = generated code is stale
```

---

## The golden corpus

CI's `golden-corpus` job runs the measurement leaves' pinned rows hermetically (no
network, no containers) across architectures, and a **version gate** refuses a leaf
version bump that has no matching moved corpus digest. Run the corpus tests locally:

```sh
docker run --rm -v "$PWD":/src -w /src golang:1.25-bookworm \
  go test ./internal/measure/...
```

The lock file lives at
`internal/measure/resolutionwalk/corpus/corpus.lock.json`. If you bump a leaf's
version, the gate expects a corresponding change to the corpus digest, the declared
parameters, or the uncovered-move rows — the rule is *output moved and the version
did not → fail*, and its inverse. See
[`docs/spec/golden-corpus.md`](../spec/golden-corpus.md).

---

## Full-stack smoke test

CI's `compose` job builds the images, brings the stack up, and waits for all three
services to report healthy. Do the same locally:

```sh
POSTGRES_PASSWORD=ci-only-password docker compose up -d --build
docker compose ps            # expect 3 services, running / healthy
docker compose logs web      # confirm migrations applied and the setup token printed
docker compose down -v       # tear down and delete volumes
```

A healthy stack here means: images build for your platform, migrations apply cleanly
against a fresh Postgres, and both `web` and `worker` pass their healthchecks.

---

## Before opening a PR — the checklist

Run these four; they are the jobs that gate merge:

1. `go vet ./...` and `go test ./...` — clean.
2. `go build` for `amd64` and `arm64` — both compile.
3. `sqlc generate` — `git diff --exit-code -- internal/db` is empty.
4. `docker compose up` — the stack reaches healthy.

There is also a `code-review` skill in this repo (`/code-review`) that reviews the
diff against the repo's documented standards and the originating issue — useful before
you push, but not a substitute for the four gates above.
