# Contributing to verge-asm

Thank you for your interest. This guide explains how to propose a change: how to
branch, how to write commits, how to run the tests, and how a pull request lands.

verge-asm follows a personal, opinionated GitHub project standard: trunk-based
branching, Conventional Commits, and green CI as the gate for every merge to `main`.

## Prerequisites

- **Go 1.25.13** (the version CI pins — see `.github/workflows/ci.yml`).
- **Docker** and the Compose plugin, to run the stack and the `compose` CI job locally.
- **sqlc 1.31.1**, if you change SQL or the schema (`sqlc generate`).

## Branching

`main` is always green and always deployable. It is protected by a ruleset. You
cannot push to it directly.

1. Cut a short-lived branch from `main`.
2. Keep it small. Merge it back within days, then delete it.
3. Land a large feature in small merges behind a flag, not one long-running branch.

Name the branch `<type>/<short-kebab-summary>`, or `<type>/<issue-#>-<summary>` when
an issue exists (preferred):

```
feat/oauth-login
fix/482-crash-on-empty-input
docs/794-doc-corpus-rewrite
```

`<type>` is one of the commit types below.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/). Each message:

```
<type>(<optional-scope>): <imperative summary, <= 72 chars>

<optional body: what and why, not how; wrap at 72>

<optional footer: BREAKING CHANGE: ..., Refs #123, Closes #123>
```

Types: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`, `chore`,
`revert`.

- One logical change per commit. Do not mix a refactor with a feature.
- `BREAKING CHANGE:` in the footer, or `!` after the type (`feat!:`), marks a
  breaking change.
- Clean up local work-in-progress commits before you open or merge a PR.

## Running the checks locally

Run these before you push. They mirror the CI jobs that gate the merge:

```sh
go vet ./...                       # static checks
go test ./...                      # unit and integration tests
sqlc generate && git diff --exit-code -- internal/db   # only if SQL/schema changed
docker compose build && docker compose up -d           # the compose stack
```

## Continuous integration

CI runs on every push and every pull request. These checks are **required** and
block the merge to `main`:

| Check | What it verifies |
|-------|------------------|
| `test` | `go vet` and `go test ./...` pass |
| `sqlc` | Generated code in `internal/db` is up to date |
| `gosec` | No new high-severity, high-confidence SAST finding |
| `govulncheck` | No reachable known-vulnerable dependency symbol |
| `gitleaks` | No secret in the git history |
| `analyze (go)` | CodeQL scan of the Go code |
| `analyze (javascript-typescript)` | CodeQL scan of the JS/TS code |

## Pull requests

Every change lands through a pull request. The PR is the review and CI gate.

- **Small and focused.** Target under ~400 lines of diff. One PR does one thing.
- **Fill in the template** (`.github/PULL_REQUEST_TEMPLATE.md`): what, why, how to
  test, screenshots for UI, and `Closes #<issue>`.
- **Use a Conventional Commits title.** The repository merges by **squash**, and
  GitHub uses the PR title as the squash commit message verbatim.
- **CI must be green** before the merge. The branch auto-deletes on merge.
- Self-review the diff first. Leave inline notes on any non-obvious choice.

## Issues

Issues live as GitHub issues. Manage them with the `gh` CLI. See
[`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md) for the conventions
and [`docs/agents/triage-labels.md`](docs/agents/triage-labels.md) for the labels.
New human-filed issues get the `needs-triage` label automatically.

## Security

Do not open a public issue for a vulnerability. Report it privately. See
[`SECURITY.md`](SECURITY.md).
