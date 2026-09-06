# Contributing to verge-asm

Thank you for your interest. This guide explains how to propose a change: how to
branch, how to write commits, how to run the tests, and how a pull request lands.

verge-asm follows a personal, opinionated GitHub project standard: trunk-based
branching, Conventional Commits, and green CI as the gate for every merge to `main`.

## Prerequisites

- **Go 1.26.8** (the version CI pins — see `.github/workflows/ci.yml`).
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

No workflow job holds both `contents: write` and `id-token: write`, and no
machine check enforces that rule (see the header of `.github/workflows/release.yml`).

## Pull requests

Every change lands through a pull request. The PR is the review and CI gate.

- **Small and focused.** Target under ~400 lines of diff. One PR does one thing.
- **Fill in the template** (`.github/PULL_REQUEST_TEMPLATE.md`): what, why, how to
  test, screenshots for UI, and `Closes #<issue>`.
- **Use a Conventional Commits title.** The repository merges by **squash**, and
  GitHub uses the PR title as the squash commit message verbatim.
- **CI must be green** before the merge. The branch auto-deletes on merge.
- Self-review the diff first. Leave inline notes on any non-obvious choice.

## Releases

A `vX.Y.Z` tag starts the release pipeline, and
[`docs/spec/release-pipeline.md`](docs/spec/release-pipeline.md) is the contract for
it. This section covers one case only: a published release turns out to be bad.

**Repair and containment are different acts.** Repair makes a good release exist.
Containment stops the bad one from reaching a new host. They use different levers and
they have different actors.

**Repair is always `vX.Y.Z+1`**, even when the fix is one character. The version never
skips, because a skipped number signals a withdrawal that `isNewer` cannot represent.

### Withdrawal: one lever permitted, two refused

| Lever | Verdict | Reason |
| --- | --- | --- |
| the **pre-release flag** | **permitted** | The one working lever. The feed skips the release and its assets stay publicly downloadable. |
| the **draft flag** | refused | A draft hides its assets. That breaks the published asset set and the air-gap kit at once. |
| **deleting the Release** | refused | The same asset loss, plus it destroys the `CHANGELOG.md` permalink. |

The SPEC refuses a pre-release **tag**. That refusal does not reach the **flag**,
because the flag names no version string.

**The `latest` GHCR tag may move backwards.** It is the only movable pointer a release
creates, and a default `docker compose pull` resolves it. One hand-run command moves
it:

```sh
docker buildx imagetools create -t <image>:latest <image>:vX.Y.Z-1
```

Add no third workflow file and no `workflow_dispatch` for this. It is an incident act
one person performs once.

**A released image tag stays pullable forever.** Deleting a GHCR package version
breaks every verify command, breaks the air-gap kit for anyone who already built one,
and breaks the digest-pinned asset for anyone who pinned it. Immutability is the
property the whole supply chain rests on. **Containment is `latest`, never deletion.**

### The incident procedure

1. Flag the Release pre-release. The feed stops serving it.
2. Move `latest` back with `imagetools create`. A default `docker compose pull` stops
   fetching the bad image.
3. Ship `vX.Y.Z+1`. Carry a `docs/release-notes/vX.Y.Z+1.md` override that states the
   withdrawal and its reason.

Containment runs first. Steps 1 and 2 are two commands, and step 3 is a full pipeline
run.

A withdrawal forces the successor onto that override route. `CHANGELOG.md` gets
nothing. `git cliff` generates it from commits, a withdrawal is not a commit, and a
hand-edited line would break the generated-output rule.

### `v0.1.0` cannot be withdrawn

This is a named exception, and it dissolves permanently once a second release exists.

Both levers fail on the first release. `/releases/latest` returns **404** when no
non-prerelease release remains. The fetcher reports that 404, `Check` logs it and
returns before it writes, so the cache stays exactly as it was. And `latest` has no
earlier tag to move back to.

Flagging `v0.1.0` pre-release is therefore **refused**. It would make every instance's
check fail silently forever. That is worse than serving a known-bad version that
`v0.1.1` replaces. **The first release is superseded, never withdrawn.**

### Why this section is not a guide page

[`docs/guides/embed.go`](docs/guides/embed.go) globs `docs/guides/*.md` only, so
`CONTRIBUTING.md` does not ride inside the shipped `web` image. An incident procedure
for the release pipeline must not be compiled into the product it repairs. The
operator half does ship, as
[`docs/guides/running.md` → Rolling back](docs/guides/running.md#rolling-back).

## Issues

Issues live as GitHub issues. Manage them with the `gh` CLI. See
[`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md) for the conventions
and [`docs/agents/triage-labels.md`](docs/agents/triage-labels.md) for the labels.
New human-filed issues get the `needs-triage` label automatically.

## Security

Do not open a public issue for a vulnerability. Report it privately. See
[`SECURITY.md`](SECURITY.md).
