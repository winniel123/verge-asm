# verge-asm

## Agent skills

### Issue tracker

Issues live as GitHub issues on `winniel123/verge-asm`. Manage them with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles. Each label string equals its name. See `docs/agents/triage-labels.md`.

### Domain docs

The repo is single-context. One `CONTEXT.md` and one `docs/adr/` sit at the repo root. See `docs/agents/domain.md`.

### Design system

All visual work uses the Verge ASM design system at `design-system/`. This covers production UI, prototypes, mocks, and slides. Invoke the `verge-asm-design` skill before you write markup. `design-system/` is the shared home for UI assets. It is the source of truth. The web app embeds and serves `templates/` and `tokens/` through `design-system/designfs.go`. The docs-site reads `tokens/` and `components/`. You may edit all of it in the repo. The old design-system handoff workflow authored markup in a separate package and byte-compared it into this repo. That workflow was retired 2026-08-28. See superseded ADR-0109 and ADR-0116. See `docs/agents/design-system.md`.
