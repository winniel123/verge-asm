# verge-asm

## Agent skills

### Issue tracker

Issues live as GitHub issues on `winniel123/verge-asm`, managed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical triage roles, each label string equal to its name. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — one `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Design system

All visual work — production UI, prototypes, mocks, slides — uses the Verge ASM design system at `design-system/`. Invoke the `verge-asm-design` skill before writing markup. `design-system/` is the shared UI asset home and the source of truth: `templates/` and `tokens/` are embedded and served by the web app (via `design-system/designfs.go`), `tokens/` and `components/` are consumed by the docs-site, and all of it may be edited in-repo. (The design-system handoff workflow — where markup was authored in a separate package and byte-compared into this repo — was retired 2026-08-28; see superseded ADR-0109 and ADR-0116.) See `docs/agents/design-system.md`.