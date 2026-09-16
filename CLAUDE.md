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

All visual work uses the Verge ASM design system at `design-system/`. This covers production UI, prototypes, mocks, and slides. Invoke the `verge-asm-design` skill before you write markup. `design-system/` is the shared home for UI assets. It is the source of truth. The web app embeds and serves `templates/` and `tokens/` through `design-system/designfs.go`. The docs-site reads `tokens/` and `components/`. You may edit all of it in the repo. The old design-system handoff workflow authored markup in a separate package and byte-compared it into this repo. That workflow was retired 2026-08-28. See ADR-0145. See superseded ADR-0109 and ADR-0116. See `docs/agents/design-system.md`.

### Model per task

The session default is Fable 5.1. Planning skills such as `/wayfinder` and `/to-tickets` inherit it. `/implement` resolves to the project skill at `.claude/skills/implement/SKILL.md`. That skill pins `model: opus`, so the implement turn runs on Opus 5. The session returns to the default on the next prompt. Run `/model opus` first when you expect several implementation prompts in one session. The plugin copy stays reachable as `/mattpocock-skills:implement`.

## Start of work

Do this before your first file change in a session. This step is mandatory. It applies to a one-line change.

1. Call the `EnterWorktree` tool. Name the worktree `<type>/<kebab-summary>`, or `<type>/<issue#>-<summary>` when an issue exists. Use a Conventional Commit type.
2. Rename the branch. `EnterWorktree` creates the branch as `worktree-<name>` and replaces `/` with `+`. That name breaks the project standard. Run `git branch -m <type>/<kebab-summary>`.
3. Confirm the branch. Run `git status -sb`. The branch must not be `main`.

The worktree is cut from `origin/main`. This satisfies the trunk-based rule in `CONTRIBUTING.md`.

A `PreToolUse` hook blocks `Edit`, `Write`, and `NotebookEdit` when the target file sits on `main`. The hook is `.claude/hooks/require-task-branch.sh`. It fails open. Do not treat a permitted edit as proof of a correct branch. Check the branch yourself.

Skip this step only for read-only work. Reading, searching, and answering a question need no branch.

Do not leave a stale worktree behind. Call `ExitWorktree` when the work is done. Use `remove` after the PR merges. Use `keep` when the work continues in a later session.

## Workflow

A typical task moves through these steps. Follow the GitHub project standard throughout. A wayfinder map is planning only. An implementation map is for implementation.

1. A bug, feature, chore, security, or doc task starts with a wayfinder chart via `/wayfinder`. The destination is always a SPEC, unless the user specifies otherwise.
2. Sessions work through the wayfinder map until the destination is complete.
3. Hand the SPEC to `/to-tickets`. Its output is a NEW parent map, separate from the closed wayfinder map. This parent has the same structure as a wayfinder map, but it is not a wayfinder map. Label that parent issue `implementation:map`. The label is what tells a later session the issue is a map and not a ticket.
4. Sessions iterate over the tickets with `/implement` until the implementation map is complete.

**One ticket per PR.** A pull request closes one ticket. This covers a fix, a feature, a chore, a security task, a doc task and an audit finding. It is not scoped to `/implement`, and it is not scoped to a map.

When you run `/implement` on an issue labelled `implementation:map`, do exactly one ticket, then stop. Do not chain the next ticket into the same session. Pick the first ticket on the frontier, implement it, open its PR, and end the session. A map with nine tickets takes nine sessions.

This rule holds even when the next ticket looks small or looks blocked on nothing. A session that runs several tickets produces one PR that mixes them, loses the per-ticket review, and buries a regression in the noise.

**File count is not the rule. Ticket count is.** A mechanical sweep with no behaviour change — one rename across many files under one ruling — is one ticket, and it may touch as many files as the rename reaches.

A measurement on the September corpus put the per-commit defect rate of a batch fix PR above a single-ticket one. Treat that figure as an **upper bound**, never as a fact: it rests on `git blame`, which names the last commit to touch a line, so a 50-file commit over-attributes. A hand-read of every batch-blamed bug found no injected code defect, because 273 of 306 September `bug` issues report pre-existing drift rather than a regression (#2199). The rule rests on the review-loss argument above and on the bookkeeping cost, not on the rate.

The bookkeeping cost is measured. PR #2074 referenced 31 issues and left 24 open (#2083). PR #2190 referenced 73 and left eight open with no verification (#2210).

`/implement` on a plain ticket number implements that ticket. Only a map argument triggers the frontier pick.

At the end of a wayfinder or implementation session, open a PR and make sure the branch is up-to-date with `main`. A human squashes and merges the PR.

### Filing a surfaced issue

Work surfaces problems that sit outside the current task. A bug, a gap, a stale doc, a missing test, a wrong comment. File each one as a GitHub issue in the same turn that surfaces it. Do not ask the user first. An unfiled finding dies with the session.

Every issue you file this way carries the `needs-triage` label.

```sh
gh issue create --label needs-triage --title "<title>" --body "<body>"
```

Add the other triage labels from `docs/agents/triage-labels.md` when the role is clear. Name the issue number in your response. Then continue the original task. Do not widen the current branch to fix what you filed.

One exception. A decision that belongs in an ADR gets no issue from you. Only a human opens that issue, because its number becomes the ADR number. Read "Writing an ADR" below.

### Writing an ADR

An ADR records one decision that passes three tests. It is hard to reverse. A reader without context would ask why. The session chose it over a named alternative. When one fails, write no ADR. Keep the reason in code, and put the rest in the PR body.

Only a human opens the issue that becomes an ADR, and its number is the ADR's number. A deleted comment that passes the three tests earns a decision proposal block in the PR body, never an issue. Under orchestration only the orchestrator writes an ADR, one PR at a time.

An ADR PR adds one file: YAML front matter, a Decision block under 150 words, a proof. A fresh-context subagent runs the `adr-review` skill, and the required `adr-review` check holds the merge. A later ADR changes an earlier one through a relation, and the tool writes the marker. Never hand-edit a marker. See `docs/spec/adr-governance.md`.

## Landing PRs on `main`

`main` is protected by an active repository RULESET, not classic branch protection. `gh api repos/.../branches/main/protection` returns a misleading 404. Check `gh api repos/winniel123/verge-asm/rulesets` instead. No direct pushes. Every change goes through a PR.

17 required status checks must pass before merge. They are `test`, `staticcheck`, `gosec`, `govulncheck`, `gitleaks`, `sqlc`, `analyze (go)`, `analyze (javascript-typescript)`, `citations`, `adr-sections`, `adr-review`, commentlint's `lint`, `corpus-version-gate`, `query-harness`, and the three `golden-corpus` legs `golden-corpus (ubuntu-24.04, v1)`, `golden-corpus (ubuntu-24.04, v3)` and `golden-corpus (ubuntu-24.04-arm, v8.0)`. `citations`, `adr-sections`, and `lint` joined on 2026-09-07 as Lane A of the ADR-drift repair. `adr-review` joined on 2026-09-08 (#1740). `corpus-version-gate` and the three `golden-corpus` legs joined on 2026-09-09. `query-harness` joined on 2026-09-16. The ruleset is the only record of those dates.

**Count this list against the ruleset before you trust it.** It said 16 and omitted `query-harness` for the whole of 2026-09-16, because a maintainer registered that check while a session was reading this file. #2255 was filed against the stale reading and asked for a promotion that had already happened. A branch then deleted the job, and the required check sat on `Expected`, which blocks every merge in the repository.

```sh
gh api repos/winniel123/verge-asm/rulesets/21255106 \
  --jq '.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context'
```

Never delete a workflow job without running that first. A required check that no workflow reports never resolves.

- `gosec` and `govulncheck` BLOCK. `govulncheck` fails on any reachable advisory. `gosec` runs `-exclude-generated -severity high -confidence high`.
- `test` runs `go vet` and `go test`. It sets no DSN, so `internal/dbtest` skips there; `query-harness` is the check that runs that tier.
- `query-harness` runs `internal/dbtest` against a `postgres` service and calls `scripts/assert-dbtest-ran.sh`, which fails the job on any `--- SKIP` and on a run with no `--- PASS` (#2255).
- `sqlc` runs `sqlc generate` then `git diff --exit-code -- internal/db`. Any migration or query change must ship regenerated `internal/db`.
- Strict up-to-date policy. When you merge PRs in sequence, update each later branch after an earlier merge. This re-triggers CI. `gh pr update-branch` does not exist in `gh` 2.45.0. Run `gh api --method PUT repos/winniel123/verge-asm/pulls/<n>/update-branch` instead.

`go.mod` pins `go 1.26.8`. `.go-version` and the Dockerfile base digest also pin 1.26.8. `.go-version` is the source of truth: every `setup-go` step reads it through `go-version-file`, and `scripts/check-go-pins.sh` runs inside the `test` job to hold the three pins together. The old CI `GO_VERSION` env key is gone (#1247). Do not use 1.27-only features. Do NOT add a `toolchain` directive equal to the `go` line — it breaks CI's `-mod=readonly` build.

New goose migrations race on their number. The `compose` CI job boots the real `web` binary, which runs `goose.Up`; a duplicate goose version panics the binary and `compose` fails at "wait for a healthy stack" (look for `panic: goose: duplicate version NNNNN`). CI tests your branch merged with `main`. Before pushing, `git fetch origin main` and number your migration above `origin/main`'s current max in `db/migrations/` (they increment by ~100).

New ADRs race on their number in the same way, and no check catches it. `docs/adr/` numbers sequentially, and every branch cuts from `origin/main`. Two concurrent branches read the same max and claim the same next number. A fetch of `origin/main` cannot reveal the clash, because neither ADR has merged. The first merge wins and the second conflicts on the file.

Before you write an ADR, read the number every open PR already claims:

```sh
gh pr list --state open --json number --jq '.[].number' \
  | xargs -I{} gh pr diff {} --name-only \
  | grep '^docs/adr/'
```

Number above `origin/main`'s current max AND above every number that command prints. State the number in your PR title or body, so a sibling session sees it without reading a diff.

On 2026-09-07 four concurrent sessions each authored ADR-0222 (#1614, #1617, #1618, #1622). Three branches then rewrote every citation of their own number. One of the three also needed a `sqlc` regeneration, because the citation sat in a SQL comment that `sqlc` lifts into `internal/db`.

## Local dev environment

The dev machine is Ubuntu Server 24.04 LTS (x86_64). It replaced a Windows machine on 2026-09-03. Every Windows-specific rule in an earlier version of this file is retired.

`sudo` asks for a password here. Do not run a command that needs root. Ask the user to run it.

### Toolchain

- **Go 1.26.8.** Install it from the go.dev tarball into `~/.local/go`, and put `~/.local/go/bin` on `PATH`. The `golang-go` apt package is 1.22 and too old. `go.mod` pins `go 1.26.8` and has no `toolchain` line. Do not use 1.27-only features.
- **Docker with the Compose plugin.** `docker.io`, `docker-buildx` and `docker-compose-v2` come from apt. The user must be in the `docker` group.
- **sqlc 1.31.1.** Do not install it. Run `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`.
- **Node 22.12.0 or newer.** `docs-site/package.json` pins `astro ^7.2.8`, and Astro 7 declares `engines.node` of `>=22.12.0`. `docs-site/package.json` declares the same floor, so a wrong Node names its own cause. Only `docs-site/` and the `doclint` and `commentlint` jobs need Node at all. The apt package is 18, so use nvm or NodeSource. Four required checks run Node: `citations`, `adr-sections` and `adr-review` in the `doclint` workflow, and `lint` in the `commentlint` workflow.
- **Node 20 stays correct for CI.** The `doclint` and `commentlint` workflows pin `node-version: 20`. Their suites are plain Node. The `docs-site` workflow builds inside a Playwright container. That container supplies its own Node above the Astro floor.

Run `go version` before you trust the toolchain. A missing `go` means the setup above is incomplete.

### Building docs-site locally

This machine holds one Node, v22.23.2, under `~/.nvm/versions/node/`. `node` is off `PATH` by default. Put `~/.nvm/versions/node/v22.23.2/bin` on `PATH` first. That version clears the 22.12.0 Astro floor, so `npm run build` succeeds here. The build takes about 10 seconds. It writes `docs-site/dist/` and `docs-site/.astro/`. `docs-site/.gitignore` ignores both directories.

A fresh worktree holds no `docs-site/node_modules`. Run `npm install` in `docs-site/`, or link that directory from the main checkout. A link is also ignored.

`npm run build` is the strongest local gate on `docs-site/`. Run it for any change to a page, a layout, or a component. Two narrower gates stay useful:

- `npm run test:doclint` runs `node --test scripts/doclint.test.mjs`. It gates the doclint script itself, and it is faster than a build.
- An `esbuild --minify` byte comparison of each touched JS file, taken before and after the edit, gates a comment sweep. `esbuild` lives in `docs-site/node_modules/.bin/`. A build does not replace it. A build proves that the site compiles, not that a comment sweep left behaviour unchanged.

### Serving a prototype

Serve every prototype on port **8090**. Always use that port. `8080` belongs to the compose stack's `web` service, and `4321` belongs to the Astro dev server. `8090` is free.

Bind the server to `127.0.0.1`. Run it from the prototype's own directory:

```sh
cd prototypes/<name> && python3 -m http.server 8090 --bind 127.0.0.1
```

Start it in the background. A foreground server blocks the session.

Two rules hold, and both are about what the port exposes:

- **Do not serve the repository root.** A server rooted at the repo publishes `.env`, `.git/`, and every source file. Serve the one prototype directory.
- **Do not bind to `0.0.0.0`.** This machine is a VPS. `0.0.0.0` publishes the directory to the internet, and the server asks for no password.

The user reaches the port through an SSH tunnel. They run this on their own machine:

```sh
ssh -L 8090:127.0.0.1:8090 <user>@<host>
```

Then they open `http://127.0.0.1:8090/` there.

One process at a time holds the port. Stop the running server before you start another.

### Running the checks

Four required checks are reproducible here: `test`, commentlint's `lint`, `citations`, and `adr-sections`. Run every one your change can reach. `staticcheck`, `gosec`, `govulncheck`, `gitleaks` and `sqlc` have no recipe in this section yet.

**The Go gates.** Part of the required `test` job, which also runs `scripts/check-go-pins.sh --mode ci`.

```sh
test -z "$(gofmt -l .)" && go vet ./... && go test ./... -count=1
```

`gofmt -l` exits 0 whether or not it names a file, so the output is the signal. A bare `gofmt -l . && …` chain always proceeds and reports success on an unformatted tree.

That chain leaves `internal/dbtest` skipped. The `test` job runs it against a database, so add `./scripts/dbtest.sh` for any change to a query, a migration, or that package.

**commentlint.** The required `lint` job. It lints the files the pull request changed, so reproduce its file set from the diff against `main`, not from `git status`:

```sh
git fetch origin main
readarray -t changed < <(git diff --name-only --diff-filter=ACMRT origin/main...HEAD)
go run ./cmd/commentlint lint --in-scope-only "${changed[@]}"
```

That is the job's own command, less `--github`. Exit 1 is a violation and exit 2 is a lex failure; both fail the job. Five facts decide whether your run matches the job's.

- **The diff, not the working tree.** A run over uncommitted files alone never lints a branch whole. Re-run the sweep after every commit. Fetch first: a worktree's `origin/main` goes stale, and a stale base inflates the file set with work that already merged.
- **Not only `.go`.** In scope: `.go`, `.mjs`, `.ts`, `.jsx`, `.tmpl`, `.css`, and `.sql` under `db/queries/`. Out: `internal/db/`, `prototypes/`, any `node_modules`, and a `.sql` file under `db/migrations/` — but a `.go` file under `db/migrations/` is in scope. `.html` and `.astro` are refused outright. A branch that touches no Go file still faces this job.
- **`.jsx` lexes through esbuild**, which lives in `docs-site/node_modules`. Without it the lexer errors and the run exits 2. Link that directory before you lint a `.jsx` file.
- **The tool names more rules than the comment policy does.** Beyond `go-decl-comment`, `change-narration`, `todo-marker`, `citation-over-one-line` and `column-over-cap`, every delete-set class is its own rule id: `short-label`, `section-divider`, `commented-out-code`, `docstring-exported-conventional`, `docstring-unexported`. Read the rule id the tool prints rather than guessing which rule you hit.
- **The 100-column cap measures the widest source line the comment block spans**, not the comment's own width. A five-character trailing comment on a 102-column code line trips `column-over-cap`, and wrapping the comment does not clear it. Declaration position stays empty, and a cited comment fits on one line, because contiguous comment lines lex as one block.

`commentlint verify --base <ref>` is a different tool with a different job. It proves a diff moved no non-comment byte, so it reports `changed` for any file that also carries a code edit. It fits a pure comment sweep and nothing else.

**`internal/dbtest` is binding in CI, and silent locally unless you give it a database.** That package executes generated queries against a real Postgres. Every case calls `dbtest.Queries(t)`, which skips when `VERGE_TEST_DATABASE_URL` is unset.

The variable is deliberately not `DATABASE_URL`: the package applies migrations and writes rows, and `DATABASE_URL` is the name a developer and every container already point at a live instance.

`query-harness` is the check that binds it. That job sets the variable against a `postgres` service, runs the tier with `-count=1 -v`, and calls `scripts/assert-dbtest-ran.sh`, which fails on any `--- SKIP` and on a run with no `--- PASS`. It is **required** as of 2026-09-16, so a case written there does block a merge.

`test` sets no DSN, so the tier skips there and the job still reports `ok`. That is deliberate: it gates the skip itself, since a case that stopped skipping would fail in `test`.

Run it locally with `./scripts/dbtest.sh`. It starts a throwaway Postgres on port 5442, runs the tier, applies the same guard script, and removes the container. Pass `go test` flags straight through, such as `./scripts/dbtest.sh -run TestMarkVantage`.

A second worktree running it at the same time needs its own port: `VERGE_DBTEST_PORT=5443 ./scripts/dbtest.sh`. The container is named after the port, so two runs on one port still collide.

Plain `go test ./...` here still skips every case and still reports `ok`. Do not read that as a pass: Go **discards a passing package's output**, so neither the skip lines nor any warning the package prints reach the terminal without `-v`. `./scripts/dbtest.sh` is the only local run that proves anything.

**The Node doc gates.** Put Node on `PATH`, keep Go on it too, and give the worktree a `docs-site/node_modules` the way "Building docs-site locally" above describes. `check:citations` shells out to `go run ./cmd/godecls` for its Go anchors, and an absent toolchain makes it exit 2 rather than return a verdict.

```sh
export PATH=$HOME/.nvm/versions/node/v22.23.2/bin:$PATH
cd docs-site
npm run -s check:citations
npm run -s check:adr-sections && npm run -s check:adr-index && npm run -s check:adr-markers
```

The `adr-sections` job runs all three of those, so `check:adr-sections` alone leaves a stale `docs/adr/index.json` or a hand-edited marker to fail the merge. Regenerate the index with `npm run write:adr-index`.

`check:citations` reads a pull-request body for its `Site` arm, so a local run reports that it judged no `Site` field and gates everything else normally. The fourth Node check, `adr-review`, has no local run at all: `npm run -s check:adr-review` exits 2 with `GITHUB_EVENT_PATH and GITHUB_REPOSITORY are required`.

Run the Go gates in the container instead when you want CI's exact image:

```sh
docker run --rm \
  -v "<absolute-path-to-worktree>:/src" \
  -v verge-gomodcache:/go/pkg/mod \
  -v verge-gobuildcache:/root/.cache/go-build \
  -w /src golang:1.26.8 \
  sh -c "go vet ./... && go test ./... -count=1"
```

The two named volumes cache the module downloads and the build output. The first run downloads the module graph and is slow. Later runs reuse both caches.

The compose stack needs a `.env` file. Copy `.env.example` and set `POSTGRES_PASSWORD`. Compose fails rather than defaulting it.

### Retired Windows traps

`.gitattributes` is `* text=auto`. Linux checks every file out LF, so the CRLF traps are gone:

- **gofmt** is trustworthy again. `gofmt -l` no longer flags almost every file. CI now gates it: the `test` job fails when `gofmt -l .` names a file (#1311). `test` is a required check, so an unformatted file blocks the merge.
- **sqlc regen** no longer rewrites line endings. `git status` after `sqlc generate` shows only the files that really changed. The `git diff --numstat` workaround is no longer needed.
- **Golden corpus tests** should pass. The CRLF golden was the cause, and it is gone. This covers `internal/measure/*/corpus` and `internal/custody/corpus`. Confirm it on the first `go test ./...` after you install Go. Treat a failure as a real regression.
- `internal/auth/TestLoadOrCreateKey` and `cmd/worker/TestExecProbeRoundTrip` failed under the retired native-Windows setup. Treat a failure in either as a real regression now.

## Comments

Write a comment only when it passes both gates:

1. **Unrecoverable** — a competent reader cannot recover the fact from the declaration, its body, and its callers.
2. **External cause** — the fact names a decision, a constraint, a hazard, a cost, or a rejected alternative outside this code.

When you cannot decide, write nothing.

A surviving comment takes the form `// <reason clause> (ADR-nnnn §x.y, #nnn)`, 25 words or fewer. The citation is optional. Put it beside the statement its reason is about. Declaration position stays empty.

A machine directive (`//go:`, `-- +goose`, `eslint`) is not a comment. Write one when the tool needs it.

Restating what the code does fails gate 1. Narrating a change ("updated to handle X") fails both. Explain a change in your response.

When you edit code, delete a comment the edit makes redundant or wrong. Write a comment I explicitly request.
