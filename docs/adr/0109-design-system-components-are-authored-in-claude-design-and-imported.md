# ADR-0109: design-system components are authored in Claude Design and imported, never built in this repo

- **Status:** Accepted
- **Date:** 2026-08-22
- **Ticket:** [#265 ADR-0109: design-system components are authored in Claude Design and imported](https://github.com/winniel123/verge-asm/issues/265)
- **Map:** [#263 Migrate to the redesigned design system (AFK)](https://github.com/winniel123/verge-asm/issues/263)

## Context

Verge ASM has just migrated its visual language: `design-system/` is now the redesigned "clean,
friendly, modern SaaS" system, and the old "engineered paper" system is frozen to
`design-system-legacy/` for the dated prototypes ([map #263](https://github.com/winniel123/verge-asm/issues/263)).
The new system arrived as **~110 already-authored components** across `forms/ display/ feedback/
navigation/ media/`, each shipping a `.jsx`, a `.d.ts` contract, and a `.prompt.md` usage note. None of
them were written in this repository. They were authored in **Claude Design** — the dedicated design
workspace — and imported wholesale. Adopting them was **import, not authoring**; the migration added
no components.

That distinction is the decision this ADR pins. A design system holds its coherence — one type scale,
one severity ramp, one elevation model, light and dark in step — only while a single authoring
authority owns the components. The moment a coding session, mid-feature, hand-rolls a `.jsx` "just this
once" because a screen needs a control the system lacks, the system forks: the new component skips the
design workspace's review, drifts from the token vocabulary, and there is no `.prompt.md` telling the
next session how to use it. Every past drift-integrity argument in this repo — the domain term winning
over the visual convention, the severity ramp staying exactly five levels — assumes the component set
is governed. An ungoverned back door defeats it.

The system already ships a `design-system/COMPONENT-REQUEST.md` handoff form for exactly this case, and
the boundary is stated in `design-system/AUTHORING.md`. What was missing was a durable, discoverable
ruling that a session cannot miss and cannot re-litigate. This ADR is that ruling.

## Decision

> **Verge ASM does not author design-system components. All components are created in Claude Design and
> imported into `design-system/`. When a screen needs a component the system does not have, do not build
> it here: write a component-request markdown file from `design-system/COMPONENT-REQUEST.md`, and hand it
> to the user to give to Claude Design. Restyling within existing tokens/components is fine; creating a
> new component file in this repo is not.**

Three clarifications on the boundary, so the line is not re-drawn each time:

1. **Restyling is not authoring.** Changing how an existing component or an existing template class
   renders — within the existing token vocabulary — is ordinary work and stays in this repo. The
   production stylesheet (`cmd/web/templates.go`'s `pageCSS`) is template-local CSS, not a component
   library; re-skinning its classes is restyling, not authoring.
2. **Importing is not authoring.** Bringing a component back from Claude Design and dropping its
   `.jsx` + `.d.ts` + `.prompt.md` into `design-system/components/` is the intended flow, not a
   violation — the authorship happened in the design workspace.
3. **A missing component is a request, not a build.** The absence of a needed component is resolved by
   filling in `design-system/COMPONENT-REQUEST.md` (saved under `design-system/requests/<name>.md`) and
   handing it to the user for Claude Design — never by creating a new component file in this repo.

## Consequences

- The rule is stated at **four enforcement sites**, so it is discoverable from wherever a session
  first touches the design system, and each cites this ADR:
  - **`CLAUDE.md`** — the repo's top-level instructions, loaded every session (one-line statement +
    pointer).
  - **The `verge-asm-design` skill** (`.claude/skills/verge-asm-design/SKILL.md`) — invoked before any
    markup, carries a prominent "never author a component here" guardrail.
  - **`docs/agents/design-system.md`** — documents the Claude Design handoff process end to end.
  - **The template** (`cmd/web/templates.go`) — the production UI, where a session most tempted to
    hand-roll a control actually works.
- Two files in `design-system/` hold the mechanism: **`COMPONENT-REQUEST.md`** (the fill-in handoff
  form) and **`AUTHORING.md`** (the boundary note a session lands on if it opens `design-system/`
  intending to add a component). A `design-system/requests/` folder holds filled-in requests.
- A session that needs a component the system lacks **stops and requests** rather than building. This
  trades immediacy for coherence: the screen waits on a round-trip through Claude Design, and in
  exchange the system never forks. Where the wait is unacceptable, the correct move is still a request
  plus an explicit note of the blockage — never a repo-authored component.
- **No component authored in this repo is grandfathered.** The migration itself added none; this ADR
  keeps it that way.
