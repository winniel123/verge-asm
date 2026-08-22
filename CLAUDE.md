# verge-asm

## Agent skills

### Issue tracker

Issues live as GitHub issues on `winniel123/verge-asm`, managed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical triage roles, each label string equal to its name. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — one `CONTEXT.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### Design system

All visual work — production UI, prototypes, mocks, slides — uses the Verge ASM design system at `design-system/`. Invoke the `verge-asm-design` skill before writing markup. Verge ASM does not author design-system components. All components are created in Claude Design and imported into `design-system/`. Missing components are requested from Claude Design via `design-system/COMPONENT-REQUEST.md`, never built in-repo (ADR-0109). See `docs/agents/design-system.md`.
