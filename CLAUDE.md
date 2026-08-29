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
