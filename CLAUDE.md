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

## Start of work

Do this before your first file change in a session. This step is mandatory. It applies to a one-line change.

1. Call the `EnterWorktree` tool. Name the worktree `<type>/<kebab-summary>`, or `<type>/<issue#>-<summary>` when an issue exists. Use a Conventional Commit type.
2. Rename the branch. `EnterWorktree` creates the branch as `worktree-<name>` and replaces `/` with `+`. That name breaks the project standard. Run `git branch -m <type>/<kebab-summary>`.
3. Confirm the branch. Run `git status -sb`. The branch must not be `main`.

The worktree is cut from `origin/main`. This satisfies the trunk-based rule in `CONTRIBUTING.md`.

A `PreToolUse` hook blocks `Edit`, `Write`, and `NotebookEdit` when the target file sits on `main`. The hook is `.claude/hooks/require-task-branch.ps1`. It fails open. Do not treat a permitted edit as proof of a correct branch. Check the branch yourself.

Skip this step only for read-only work. Reading, searching, and answering a question need no branch.

Do not leave a stale worktree behind. Call `ExitWorktree` when the work is done. Use `remove` after the PR merges. Use `keep` when the work continues in a later session.

## Workflow

A typical task moves through these steps. Follow the GitHub project standard throughout. A wayfinder map is planning only. An implementation map is for implementation.

1. A bug, feature, chore, security, or doc task starts with a wayfinder chart via `/wayfinder`. The destination is always a SPEC, unless the user specifies otherwise.
2. Sessions work through the wayfinder map until the destination is complete.
3. Hand the SPEC to `/to-tickets`. Its output is a NEW parent map, separate from the closed wayfinder map. This parent has the same structure as a wayfinder map, but it is not a wayfinder map. Label that parent issue `implementation:map`. The label is what tells a later session the issue is a map and not a ticket.
4. Sessions iterate over the tickets with `/implement` until the implementation map is complete.

**One ticket per session.** When you run `/implement` on an issue labelled `implementation:map`, do exactly one ticket, then stop. Do not chain the next ticket into the same session. Pick the first ticket on the frontier, implement it, open its PR, and end the session. A map with nine tickets takes nine sessions.

This rule holds even when the next ticket looks small or looks blocked on nothing. A session that runs several tickets produces one PR that mixes them, loses the per-ticket review, and buries a regression in the noise.

`/implement` on a plain ticket number implements that ticket. Only a map argument triggers the frontier pick.

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

**There is no local Go toolchain on this machine.** Measured 2026-08-31: `go` is absent from `PATH`, absent from the persisted user and machine `PATH` in the registry, and a full `C:` drive search finds no `go.exe`. Do not spend a session hunting for it. An earlier version of this file described a local 1.26.5 toolchain. That description is retired.

Run every Go command in Docker instead. The container pins CI's exact version, so it is a stronger check than a local toolchain was:

```sh
docker run --rm \
  -v "<absolute-path-to-worktree>:/src" \
  -v verge-gomodcache:/go/pkg/mod \
  -v verge-gobuildcache:/root/.cache/go-build \
  -w /src golang:1.25.13 \
  sh -c "go vet ./... && go test ./... -count=1"
```

The two named volumes cache the downloaded modules and the build output. The first run downloads the module graph and is slow. Later runs reuse both caches. `cmd/web` alone takes about 70 s.

`go.mod` pins `go 1.25.13` and has no `toolchain` line. Pin the image tag to `golang:1.25.13` so the container never drifts above it. Do not use 1.26-only features.

`.gitattributes` is `* text=auto` (files stored LF, checked out CRLF on Windows). This causes two traps:

- **gofmt:** `gofmt -l` flags almost every file because of CRLF. This is not a real problem — git normalizes to LF on commit and CI sees clean files. Do not chase it. Run gofmt in the container, never against the Windows working tree, and never `gofmt -w` blindly.
- **sqlc regen:** CI pins sqlc v1.31.1. Run it in the container with `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`. sqlc rewrites every `internal/db/*.sql.go` with LF, so `git status` shows ~40 files modified but `git diff` shows no content hunks for the untouched ones. Find real changes with `git diff --numstat -- internal/db/` (non-zero counts changed). Restore the noise with `git checkout -- internal/db/`, keep the real files, and stage explicitly (never `git add -A`).

Pre-existing test failures in the container (NOT regressions): every `TestCorpusExpectation` in a golden corpus — `internal/measure/*/corpus` and `internal/custody/corpus`. The cause is the CRLF golden, not the host OS. The container bind-mounts the Windows working tree, so it reads the same CRLF files a native Windows run reads. CI passes them because CI checks out fresh with LF. A corpus you just blessed with `-update` in the container passes until git rewrites its goldens with CRLF on a later checkout.

`internal/auth/TestLoadOrCreateKey` (file mode) and `cmd/worker/TestExecProbeRoundTrip` (prober not in `PATH`) failed under the retired native-Windows setup. Both pass in the container. Treat a failure in either as a real regression now.
