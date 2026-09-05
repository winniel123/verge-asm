# ADR-0110: the design-system examples are the console's IA spec, ported verbatim

- **Status:** Accepted
- **Date:** 2026-08-22
- **Ticket:** [#276 T0 Foundation](https://github.com/winniel123/verge-asm/issues/276)
- **Map:** [#275 Migrate the product console to the V2 design-system contract (AFK)](https://github.com/winniel123/verge-asm/issues/275)
- **Supersedes:** the "`examples/` is a reference look, not an IA ruling" clause of `docs/agents/design-system.md` — and only that clause.

## Context

When the redesigned system landed ([map #263](https://github.com/winniel123/verge-asm/issues/263),
[ADR-0109](0109-design-system-components-are-authored-in-claude-design-and-imported.md)), one guardrail
in `docs/agents/design-system.md` held the `examples/` at arm's length: they were "reference
compositions, not IA decisions", to be rebuilt in structure rather than shipped verbatim. That caution
was correct for a system whose information architecture the domain still owned and whose examples might
carry a design-workspace reading of the product that diverged from the domain — most sharply, a
dashboard-first example could quietly demote drift, the one screen this product exists to draw.

Two things have since changed. First, the V2 handoff `README.md` restates the examples' status
explicitly: they "are the screens themselves — the spec, not inspiration", ported verbatim (same
components, layout, spacing, hierarchy, copy), swapping only inline sample data for real data of the
same shape and the local `screen` state for the app's router. Second — and this is what makes the
reversal safe — the canonical console (`examples/console/ConsoleApp.jsx`) is **already
domain-correct**: its seven top-level screens are **Dashboard · Scope · Inventory · Drift · Signals ·
Graph · Reports**, with Drift as nav item 4 of 7, plus Settings and SignIn. The example does not demote
drift. It carries drift as a first-class screen. The very risk the old clause guarded against is not
present in the artefact the clause was guarding against.

So the arm's-length posture now costs more than it protects. Rebuilding each screen's structure "from
scratch against real routing and data" reintroduces exactly the drift from the spec that
pixel-for-pixel fidelity is meant to eliminate, and does so screen by screen across a dozen tickets,
where a verbatim port is both cheaper and more faithful. The user ratified the reversal on 2026-08-22.

## Decision

> **The console screens in `design-system/examples/console/` are the console's information-architecture
> spec, ported verbatim.** Each screen's composition — its components, layout, spacing, hierarchy, and
> copy — is translated from the reference JSX into the app's server-rendered Go templates
> (`cmd/web/`), swapping only inline sample data for real data of the same shape (design-system
> empty-states where a new screen has no backing data yet, never fabricated data). `ConsoleApp.jsx` is
> the shell spec (TopNav, org switcher, ⌘K palette, toast stack, theme toggle, messages bell);
> `screenshots/` are the visual ground truth every screen verifies against.

This reverses exactly one clause and nothing else. Every other guardrail in
`docs/agents/design-system.md` stands unchanged:

- **Drift is the thesis.** Drift remains a top-level screen — nav item 4 of 7 — so "drift is never a
  secondary screen" is preserved *by* the port, not despite it. This is the fact that made the reversal
  admissible.
- **`signal` never `finding`.** Signals are **withdrawn** by the world, never "resolved" by operators.

  > **Amended 2026-09-05 by [ADR-0147](0147-assets-watched-is-the-distinct-subject-count-over-the-open-span-corpus.md) /
  > [#1288](https://github.com/winniel123/verge-asm/issues/1288): this rule refuses a KPI, not only a
  > word.** The reference mock's **mean time-to-resolve** is refused, and the reporting KPI is a **mean
  > time-to-withdrawal** (`drift.MeanTimeToWithdrawal`, rendered as `MTTW` on Reports). A resolve time
  > would have to be measured from a departure the world caused, so the number would count subject
  > departures and label them operator resolutions — an operator act the product never observed. The
  > interval is the subject's earliest `opened_at` to the closure of its timelines
  > ([ADR-0082](0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md)),
  > read off the never-compacted span corpus
  > ([ADR-0041](0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md)), so it is
  > measured rather than fabricated. This refusal was previously written only in a code comment and in
  > `SPEC-CHANGE.md`, which no longer exists in the tree.
  > [ADR-0116](0116-the-design-package-is-normative-for-look-and-functionality.md) §4 restates the
  > vocabulary rule, but it is **Superseded (2026-08-28)** and it never named the KPI either. This
  > Decision is the rule's live site.
- **Domain nouns, not wire nouns** — `Name` / `Address` / `Service` / `Endpoint`, never
  host / IP / port / URL as modelled things.
- **`seed` / `scope`** never target · **`channel`** never webhook · **`vantage`** never probe ·
  **`annotation`** never mute / triage.
- **No technology fingerprinting.**
- **Severity is exactly `Critical / High / Medium / Low / Info`** via `SeverityBadge`. The change
  vocabulary rides its own drift palette, never the severity ramp.
- ~~**Verge ASM does not author design-system components** ([ADR-0109](0109-design-system-components-are-authored-in-claude-design-and-imported.md)).
  Porting a screen translates the reference composition into template CSS classes within the existing
  token vocabulary — restyling, not authoring. A genuine component gap is a
  `design-system/requests/*.md` COMPONENT-REQUEST plus a flag, never an in-repo build.~~

  > **This bullet is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
  > [#1410](https://github.com/winniel123/verge-asm/issues/1410)
  > ([ADR-0058](0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** That
  > commit retired the design-system handoff workflow and superseded
  > [ADR-0109](0109-design-system-components-are-authored-in-claude-design-and-imported.md).
  > `design-system/` is the source of truth and may be edited in the repo — templates, tokens and
  > components alike. `design-system/COMPONENT-REQUEST.md` and `design-system/requests/` are deleted,
  > so the mechanism this bullet prescribes no longer exists. Read `CLAUDE.md` and
  > [`docs/agents/design-system.md`](../agents/design-system.md) for the rule that replaces it. The
  > rest of this Decision stands: the console screens are still the IA spec, ported verbatim.

The scope of the port is the console only (`cmd/web`). The marketing `Homepage.jsx` and `DocsPage.jsx`
have no serving surface in-repo and are out of scope.

## Consequences

- `docs/agents/design-system.md` is amended at two sites (the `examples/` bullet and the "Drift is the
  thesis" guardrail) to point here and mark the clause superseded. The system remains authoritative on
  *how things look and how copy sounds*. The examples are now also authoritative on *how the console is
  organised*, because the two no longer diverge.
- The migration (map #275) ports each canonical screen verbatim rather than reinterpreting it. New
  screens with no backing data (Drift's timeline, Graph, Reports beyond exposure) ship design-system
  empty-states, never fabricated data.
- Fidelity is checked against `design-system/screenshots/` per screen, not against a re-derived
  structure. A screen that reads as a re-composition of the example, rather than a translation of it,
  is a regression under this ADR.
- The reversal is deliberately narrow. Any *other* divergence between an example and a domain term is
  still resolved in the domain term's favour — the port translates the example's look and structure,
  not a rejected vocabulary, into the interface.
