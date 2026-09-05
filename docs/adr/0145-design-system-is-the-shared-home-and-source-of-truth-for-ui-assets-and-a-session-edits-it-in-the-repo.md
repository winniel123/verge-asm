# ADR-0145: `design-system/` is the shared home and source of truth for UI assets, and a session edits it in the repo

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1269 ADR gaps: design-system, cmd/prober and the measure corpora](https://github.com/winniel123/verge-asm/issues/1269) §1
- **Found by:** [#1163 chore(comments): sweep the Go tail 2/14](https://github.com/winniel123/verge-asm/issues/1163)
- **States the standing rule that** [ADR-0109](./0109-design-system-components-are-authored-in-claude-design-and-imported.md) and [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md) carry only in a **Superseded** status line. Both remain superseded and neither is revived.
- **Amends one bullet of:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md), whose withdrawn component-authoring bullet now cites this ADR. ADR-0110's verbatim-port ruling and its IA-spec ruling are untouched and stand.
- **Rests on:** [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) (a superseded mechanism is withdrawn at the site that specifies it)

## Context

On 2026-08-28, commit `55aa367` ([#781](https://github.com/winniel123/verge-asm/issues/781)) retired the
design-system handoff workflow. Under that workflow a session could not author UI markup here at all:
components were authored in Claude Design, imported wholesale, and byte-compared into this repo, and a
genuine gap went out as a `design-system/COMPONENT-REQUEST.md` round-trip. After the retirement,
`design-system/` is the shared home for UI assets and a session edits it in the repo.

**No Accepted ADR says so.** ADR-0109 and ADR-0116 are both **Superseded (2026-08-28)** and carry the
replacement rule only inside their status lines. A superseded ADR is not a statement of the standing
rule, and a reader who obeys a status line is reading the thing the status line retires.

The rule did have one durable statement in code. The `designfs` package doc named `design-system/` the
shared UI asset home, named both consumers, and said the artifacts may be edited in-repo because the
byte-compare gates were retired. Commit `7c6e96e` ([#1268](https://github.com/winniel123/verge-asm/issues/1268))
compressed that doc to the three lines that explain the package's location, and the rule went with it.
The deleted block cited no ADR, no issue and no spec, so nothing else in the tree carried it.

The mechanism the old workflow prescribed is gone from disk. `design-system/COMPONENT-REQUEST.md`,
`design-system/AUTHORING.md` and `design-system/requests/` do not exist. `design-system/` holds
`components/`, `docs/`, `examples/`, `fixtures/`, `templates/`, `tokens/`, `styles.css` and `README.md`.

Both consumers are load-bearing and both read the same tree:

- **The web app embeds it.** `design-system/designfs.go:11` is `//go:embed templates/*.tmpl tokens/*.css fixtures/*.json`,
  and `designfs.FS` is the only file system `cmd/web` parses templates from. `cmd/web/templates_inventory.go:17`
  globs `tokens/*.css` and concatenates every match into the served stylesheet, so a token file that is
  added, renamed or deleted changes what ships.
- **The docs-site reads it.** `docs-site/astro.config.mjs:31` aliases `@ds` to the `design-system`
  root and `docs-site/tsconfig.json:8` maps `@ds/*` to `../design-system/*`.
  `docs-site/src/styles/global.css` imports six files from `@ds/tokens/`, and
  `docs-site/src/components/TopNav.jsx` imports components from `@ds/components/`.

So one tree feeds a Go binary and an Astro site. That is why ownership needs a live statement rather
than three superseded ones.

## Decision

> **`design-system/` is the shared home and the source of truth for UI assets. The web app embeds
> `templates/` and `tokens/` through `design-system/designfs.go`; the docs-site reads `tokens/` and
> `components/` through its `@ds` alias. A session may edit all of it in the repo — templates, tokens,
> and components alike. There is no separate authoring package and no round-trip.**

### 1. One tree, two consumers, no second copy

A UI asset lives in `design-system/` and nowhere else. Neither consumer keeps a fork, a vendored copy,
or a mirrored token file. A change to a token or a template is visible to both consumers at their next
build, which is the property the shared home exists to give.

`design-system/designfs.go` is sibling glue that only **reads** the artifacts. Editing it is not
editing the design system.

### 2. Editing is ordinary work

A session that needs a component change makes it here. It does not file a request, does not raise a
flag, and does not wait on an external authoring authority. The byte-compare gates that made an
in-repo edit a violation were retired with the workflow that needed them.

### 3. Coherence is held by rules, not by a gate

The retired handoff bought coherence with a round-trip. The replacement buys it with three standing
obligations, already written in [`docs/agents/design-system.md`](../agents/design-system.md) and the
`verge-asm-design` skill: reuse before you add, stay in the token vocabulary, and ship the `.jsx` /
`.d.ts` / `.prompt.md` trio with a new component.

### 4. This ADR settles ownership only

It rules on **who may edit `design-system/` and where the assets live**. It does not rule on what the
`examples/` mean — that is ADR-0110, which stands — and it does not revive any clause of ADR-0109 or
ADR-0116. In particular the ADR-0116 questions that
[#1288](https://github.com/winniel123/verge-asm/issues/1288) and
[#1300](https://github.com/winniel123/verge-asm/issues/1300) track stay open.

## Consequences

- **ADR-0110's withdrawn component-authoring bullet gains this citation.** Its withdrawal note sent the
  reader to `CLAUDE.md` and `docs/agents/design-system.md` because no ADR held the replacement rule.
  One does now, so a reader of ADR-0110 alone reaches it.
- **The three sites that cite only the superseded pair now cite this ADR too** — `CLAUDE.md`,
  [`docs/agents/design-system.md`](../agents/design-system.md), and
  `.claude/skills/verge-asm-design/SKILL.md`. ADR-0109 and ADR-0116 stay in each citation, marked
  superseded, because they hold the history of what was retired.
- **No Go code and no markup change.** Every arrangement this ADR states already exists.
- **`design-system/designfs.go` gets no restored comment.** The package doc's compression is what
  exposed the gap, and an ADR is the durable site for a rule about ownership. The `designfs` doc
  explains the package.
- **`CONTEXT.md` gains nothing.** `design-system/` is a repository location, not a domain term.
- **A future move of the tree is an ADR, not a refactor.** Two build systems resolve paths into
  `design-system/` by name, so relocating it breaks both silently.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Keep the separate-package handoff**: author markup in Claude Design, import it wholesale, byte-compare it into this repo, and file a `COMPONENT-REQUEST.md` for a gap | Retired 2026-08-28 by `55aa367` ([#781](https://github.com/winniel123/verge-asm/issues/781)). [ADR-0109](./0109-design-system-components-are-authored-in-claude-design-and-imported.md) and [ADR-0116](./0116-the-design-package-is-normative-for-look-and-functionality.md) are **Superseded**, and the mechanism is gone from disk: `COMPONENT-REQUEST.md`, `AUTHORING.md` and `requests/` no longer exist. A rule whose apparatus has been deleted cannot be obeyed |
| **Un-supersede ADR-0109 or ADR-0116** so an Accepted ADR states the rule | They state the **opposite** rule. Their status lines are the correction, not the ruling. Reviving either resurrects the parity gates and the round-trip with it |
| **Restore the deleted `designfs` package doc** and call the gap closed | A package doc is the wrong site for a rule about repository ownership: it is invisible to the docs-site half, and the next comment sweep deletes it again on exactly the grounds that deleted it this time |
| **Rule it in `CLAUDE.md` and `docs/agents/design-system.md` only**, as today | Both already say it, and the gap persisted anyway. Agent guidance restates decisions; it is not where a decision is made, and neither file records why the handoff lost |
| **Supersede [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) whole** | Its verbatim-port and IA-spec rulings are live and are the standing contract for the console migration. Only one bullet was ever about component authoring. Under [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md) the withdrawal belongs at that bullet |
| **Also rule on ADR-0116's unsettled Decision clauses 1–3** while the subject is open | A different question, tracked by [#1288](https://github.com/winniel123/verge-asm/issues/1288) and [#1300](https://github.com/winniel123/verge-asm/issues/1300). Answering it here would settle by proximity what no one has argued |
| **Let each consumer vendor its own copy** of the tokens and components it uses | Two copies drift, and the drift shows up as a docs-site and console that disagree on the severity ramp. The shared home is the whole point |
