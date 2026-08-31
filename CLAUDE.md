# verge-asm

## Agent skills

### Issue tracker

Issues live as GitHub issues on `winniel123/verge-asm`. Manage them with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles. Each label string equals its name. See `docs/agents/triage-labels.md`.

### Domain docs

The repo is single-context. One `CONTEXT.md` and one `docs/adr/` sit at the repo root. See `docs/agents/domain.md`.

### GitHub project standard

verge-asm follows Logan's portable GitHub project standard. It governs branching, commits, CI, security, and releases. Key rules: trunk-based short-lived branches cut from `main`, Conventional Commits, and squash-only merges. `main` is ruleset-protected. Every change lands through a pull request with green CI. Branch names use `<type>/<kebab-summary>`, or `<type>/<issue#>-<summary>` when an issue exists. The repo does not yet have release automation, signed-commit enforcement, or deploy environments. Verify a settings-level item before you trust it. See `CONTRIBUTING.md`.

### Design system

All visual work uses the Verge ASM design system at `design-system/`. This covers production UI, prototypes, mocks, and slides. Invoke the `verge-asm-design` skill before you write markup. `design-system/` is the shared home for UI assets. It is the source of truth. The web app embeds and serves `templates/` and `tokens/` through `design-system/designfs.go`. The docs-site reads `tokens/` and `components/`. You may edit all of it in the repo. The old design-system handoff workflow authored markup in a separate package and byte-compared it into this repo. That workflow was retired 2026-08-28. See superseded ADR-0109 and ADR-0116. See `docs/agents/design-system.md`.

## Workflow

A typical task moves through these steps. Follow the GitHub project standard throughout. A wayfinder map is planning only. An implementation map is for implementation.

1. A bug, feature, chore, security, or doc task starts with a wayfinder chart via `/wayfinder`. The destination is always a SPEC, unless the user specifies otherwise.
2. Sessions work through the wayfinder map until the destination is complete.
3. Hand the SPEC to `/to-tickets`. Its output is a NEW parent map, separate from the closed wayfinder map. This parent has the same structure as a wayfinder map, but it is not a wayfinder map.
4. Sessions iterate over the tickets with `/implement` until the implementation map is complete.

At the end of a wayfinder or implementation session, open a PR and make sure the branch is up-to-date with `main`. A human squashes and merges the PR.

## Landing PRs on `main`

`main` is protected by an active repository RULESET, not classic branch protection. `gh api repos/.../branches/main/protection` returns a misleading 404. Check `gh api repos/winniel123/verge-asm/rulesets` instead. No direct pushes. Every change goes through a PR.

7 required status checks must pass before merge: `test`, `gosec`, `govulncheck`, `gitleaks`, `sqlc`, `analyze (go)`, `analyze (javascript-typescript)`.

- `gosec` and `govulncheck` BLOCK. `govulncheck` fails on any reachable advisory. `gosec` runs `-exclude-generated -severity high -confidence high`.
- `test` runs `go vet` and `go test`.
- `sqlc` runs `sqlc generate` then `git diff --exit-code -- internal/db`. Any migration or query change must ship regenerated `internal/db`.
- Strict up-to-date policy. When you merge PRs in sequence, update each later branch (`gh pr update-branch <n>`) after an earlier merge. This re-triggers CI.

`go.mod` pins `go 1.25.13`. CI `GO_VERSION` and the Dockerfile base digest also pin 1.25.13. Do not use 1.26-only features. Do NOT add a `toolchain` directive equal to the `go` line — it breaks CI's `-mod=readonly` build.

New goose migrations race on their number. The `compose` CI job boots the real `web` binary, which runs `goose.Up`; a duplicate goose version panics the binary and `compose` fails at "wait for a healthy stack" (look for `panic: goose: duplicate version NNNNN`). CI tests your branch merged with `main`. Before pushing, `git fetch origin main` and number your migration above `origin/main`'s current max in `db/migrations/` (they increment by ~100).

## Windows local dev gotchas

This machine's Go toolchain is 1.26.5, but CI pins 1.25.13. `go.mod` has no `toolchain` line, so `GOTOOLCHAIN=auto` lets local 1.26.5 satisfy it; local builds and tests run under 1.26.5, CI under 1.25.13.

`.gitattributes` is `* text=auto` (files stored LF, checked out CRLF on Windows). This causes two traps:

- **gofmt:** `gofmt -l` flags almost every file because of CRLF. This is not a real problem — git normalizes to LF on commit and CI sees clean files. Do not chase it. Do not `gofmt -w` blindly: local 1.26.5 gofmt rewrites to 1.26 style. To verify a file is CI-clean, check its LF-normalized content under a 1.25 gofmt.
- **sqlc regen:** CI pins sqlc v1.31.1. Run it with `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`. sqlc rewrites every `internal/db/*.sql.go` with LF, so `git status` shows ~40 files modified but `git diff` shows no content hunks for the untouched ones. Find real changes with `git diff --numstat -- internal/db/` (non-zero counts changed). Restore the noise with `git checkout -- internal/db/`, keep the real files, and stage explicitly (never `git add -A`).

Pre-existing Windows-only test failures (NOT regressions): `internal/auth/TestLoadOrCreateKey` (file mode), `cmd/worker/TestExecProbeRoundTrip` (prober not in PATH), and every `internal/measure/*/corpus` `TestCorpusExpectation` (CRLF vs LF golden). CI on Linux passes them.
