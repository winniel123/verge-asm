# ADR-0109: design-system components are authored in Claude Design and imported, never built in this repo

- **Status:** Superseded (2026-08-28) — Design-system handoff workflow retired; the repo's served templates and design-system/ assets are the source of truth. Templates and components may be edited in-repo.
- **Date:** 2026-08-22
- **Ticket:** [#265 ADR-0109: design-system components are authored in Claude Design and imported](https://github.com/winniel123/verge-asm/issues/265)
- **Map:** [#263 Migrate to the redesigned design system (AFK)](https://github.com/winniel123/verge-asm/issues/263)

## Context

Verge ASM has just migrated its visual language: `design-system/` is now the redesigned "clean,
friendly, modern SaaS" system, and the old "engineered paper" system is frozen to
`prototypes/design-system-legacy/` for the dated prototypes ([map #263](https://github.com/winniel123/verge-asm/issues/263)).
The new system arrived as **~110 already-authored components** across `forms/ display/ feedback/
navigation/ media/`, each shipping a `.jsx`, a `.d.ts` contract, and a `.prompt.md` usage note. None of
them were written in this repository. They were authored in **Claude Design** — the dedicated design
workspace — and imported wholesale. Adopting them was **import, not authoring**. The migration added
no components.

That distinction is the decision this ADR pins. A design system holds its coherence — one type scale,
one severity ramp, one elevation model, light and dark in step — only while a single authoring
authority owns the components. The moment a coding session, mid-feature, hand-rolls a `.jsx` "just this
once" because a screen needs a control the system lacks, the system forks: the new component skips the
design workspace's review, drifts from the token vocabulary, and there is no `.prompt.md` telling the
next session how to use it. Every past drift-integrity argument in this repo — the domain term winning
over the visual convention, the severity ramp staying exactly five levels — assumes the component set
is governed. An ungoverned back door defeats it.

~~The system already ships a `design-system/COMPONENT-REQUEST.md` handoff form for exactly this case, and
the boundary is stated in `design-system/AUTHORING.md`.~~ What was missing was a durable, discoverable
ruling that a session cannot miss and cannot re-litigate. This ADR is that ruling.

> **The struck `COMPONENT-REQUEST.md` and `AUTHORING.md` citations are WITHDRAWN at the site that
> states them, 2026-08-28 by `55aa367` /
> [#1453](https://github.com/winniel123/verge-asm/issues/1453)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> That commit retired the design-system handoff workflow and deleted both files. Neither path is on
> disk, so nothing holds the form or the boundary note.
> [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)
> states the rule that replaced the workflow, and [`docs/agents/design-system.md`](../agents/design-system.md)
> restates it. The sentence stays as an accurate record of what this ADR read in 2026-08.

## Decision

> **Verge ASM does not author design-system components. All components are created in Claude Design and
> imported into `design-system/`. When a screen needs a component the system does not have, do not build
> it here: ~~write a component-request markdown file from `design-system/COMPONENT-REQUEST.md`, and hand it
> to the user to give to Claude Design.~~ Restyling within existing tokens/components is fine; creating a
> new component file in this repo is not.**
>
> **The struck component-request clause is WITHDRAWN at the site that specifies it, 2026-08-28 by
> `55aa367` / [#1453](https://github.com/winniel123/verge-asm/issues/1453)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> The status line above withdraws the workflow by name. ADR-0058 rules that the unit is the sentence,
> so the clause carries its own marker. `design-system/COMPONENT-REQUEST.md` no longer exists, so this
> clause names a form nobody can fill in and a round-trip nobody can take. A session that needs a
> component today writes it in `design-system/components/`
> ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md),
> [`docs/agents/design-system.md`](../agents/design-system.md)).

Three clarifications on the boundary, so the line is not re-drawn each time:

1. **Restyling is not authoring.** Changing how an existing component or an existing template class
   renders — within the existing token vocabulary — is ordinary work and stays in this repo. ~~The
   production stylesheet (`cmd/web/templates.go`'s `pageCSS`) is template-local CSS, not a component
   library.~~ Re-skinning its classes is restyling, not authoring.

   > **The struck `pageCSS` sentence is WITHDRAWN at the site that states it, 2026-08-22 by
   > `2678e88` / [#1453](https://github.com/winniel123/verge-asm/issues/1453)
   > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
   > That commit split `cmd/web/templates.go` into seven `cmd/web/templates_{shell,error,inbox,inventory,reportartifact,reports,settings}.go`
   > files, on the same day this ADR was dated. No `pageCSS` symbol survives anywhere in the tree, so
   > the pointer names neither a file nor a symbol. The claim it carried has also reversed. The served
   > stylesheet is `designTokensCSS` in `cmd/web/templates_inventory.go`, which globs
   > `design-system/tokens/*.css` and concatenates every match, and `cmd/web/templates_shell.go`
   > injects it through the `designTokens` template func. That stylesheet is design-owned, not
   > template-local
   > ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)).
   > The rest of this clarification stands on its first sentence.
2. **Importing is not authoring.** Bringing a component back from Claude Design and dropping its
   `.jsx` + `.d.ts` + `.prompt.md` into `design-system/components/` is the intended flow, not a
   violation — the authorship happened in the design workspace.
3. ~~**A missing component is a request, not a build.** The absence of a needed component is resolved by
   filling in `design-system/COMPONENT-REQUEST.md` (saved under `design-system/requests/<name>.md`) and
   handing it to the user for Claude Design — never by creating a new component file in this repo.~~

   > **This clarification is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
   > [#1453](https://github.com/winniel123/verge-asm/issues/1453)
   > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
   > `design-system/COMPONENT-REQUEST.md` and `design-system/requests/` went with the workflow, so a
   > missing component has nowhere to go as a request. A missing component is a build, made here.
   > [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)
   > states that rule. [`docs/agents/design-system.md`](../agents/design-system.md)
   > carries the three obligations a new component owes instead — reuse before you add, stay in the
   > token vocabulary, and ship the `.jsx` / `.d.ts` / `.prompt.md` trio.

## Consequences

- The rule is stated at **four enforcement sites**, so it is discoverable from wherever a session
  first touches the design system, and each cites this ADR:
  - **`CLAUDE.md`** — the repo's top-level instructions, loaded every session (one-line statement +
    pointer).
  - **The `verge-asm-design` skill** (`.claude/skills/verge-asm-design/SKILL.md`) — invoked before any
    markup, carries a prominent "never author a component here" guardrail.
  - **`docs/agents/design-system.md`** — ~~documents the Claude Design handoff process end to end.~~

    > **The struck description is WITHDRAWN at the site that states it, 2026-08-28 by `55aa367` /
    > [#1453](https://github.com/winniel123/verge-asm/issues/1453)
    > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
    > That commit retired the handoff, and the file stopped documenting it. Its
    > §"Adding or editing a component (in-repo)" tells a session to edit `design-system/` here, and
    > cites
    > [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)
    > for the standing rule. The file still names this ADR, marked superseded.

  - ~~**The template** (`cmd/web/templates.go`) — the production UI, where a session most tempted to
    hand-roll a control actually works.~~

    > **This enforcement site is WITHDRAWN at the site that states it, 2026-08-22 by `2678e88` /
    > [#1453](https://github.com/winniel123/verge-asm/issues/1453)
    > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
    > `cmd/web/templates.go` no longer exists. `2678e88` split it into seven `cmd/web/templates_*.go`
    > files, and none of them carries a component-authoring guardrail. The rule this bullet points at
    > fell with the workflow in any case, 2026-08-28 by `55aa367`
    > ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)).
- ~~Two files in `design-system/` hold the mechanism: **`COMPONENT-REQUEST.md`** (the fill-in handoff
  form) and **`AUTHORING.md`** (the boundary note a session lands on if it opens `design-system/`
  intending to add a component). A `design-system/requests/` folder holds filled-in requests.~~

  > **This bullet is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
  > [#1453](https://github.com/winniel123/verge-asm/issues/1453)
  > ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
  > That commit removed all three paths. `design-system/` holds `components/`, `docs/`, `examples/`,
  > `fixtures/`, `templates/`, `tokens/`, `styles.css` and `README.md`. No mechanism file sits among
  > them, and nothing anywhere in the repo carries a handoff form
  > ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)).
- A session that needs a component the system lacks **stops and requests** rather than building. This
  trades immediacy for coherence: the screen waits on a round-trip through Claude Design, and in
  exchange the system never forks. Where the wait is unacceptable, the correct move is still a request
  plus an explicit note of the blockage — never a repo-authored component.
- **No component authored in this repo is grandfathered.** The migration itself added none. This ADR
  keeps it that way.
