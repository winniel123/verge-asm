<!--
  Keep this PR small and focused. One logical change.
  Target under ~400 lines of diff. Use a Conventional Commits title,
  because the squash merge uses the PR title verbatim.
  Example: fix(prober): stop retry loop on a closed port
-->

## What

<!-- What does this PR change? One or two sentences. -->

## Why

<!-- Why is this change needed? Link the problem, not the diff. -->

## How to test

<!-- The exact steps or commands a reviewer runs to verify this. -->

## Screenshots

<!-- For any UI change, add before/after screenshots. Delete this section if not a UI change. -->

## Checklist

- [ ] `go vet ./...` and `go test ./...` pass locally.
- [ ] `sqlc generate` produces no diff (if the SQL or schema changed).
- [ ] The commit history is clean (Conventional Commits, one logical change per commit).
- [ ] Docs and `CONTEXT.md` updated where behaviour changed.

Closes #
