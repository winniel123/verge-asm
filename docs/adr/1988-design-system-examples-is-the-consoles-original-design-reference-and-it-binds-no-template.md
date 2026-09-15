---
number: 1988
title: "`design-system/examples/` is the console's original design reference, and it binds no template"
slug: design-system-examples-is-the-consoles-original-design-reference-and-it-binds-no-template
date: 2026-09-15
status: accepted
source: grilling
ticket: 1988
proof: {none: "a ruling about the status of a document, which no test and no measurement can prove"}
relations:
  - {kind: amends, adr: 110}
  - {kind: rests-on, adr: 145}
---

# ADR-1988: `design-system/examples/` is the console's original design reference, and it binds no template

## Decision

> **`design-system/examples/` is the console's original design reference.** It records what each
> screen looked like when the T0 port consumed it. It binds no template, no ticket and no session.
>
> A console screen's information architecture is decided in its SPEC under `docs/spec/`, and the
> served `design-system/templates/*.tmpl` is the surface that realises it. An example may be edited
> when something else requires it — the `--console-surface` invariant in
> `design-system/console_tokens_test.go` does — but no template change obliges a matching example
> change, and a difference between the two is not a defect.
>
> Rejected: parity as a standing obligation, which no check enforces and which twelve screens
> already fail.

## 1. Context

[ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md) ruled the
console examples the information-architecture spec, ported verbatim. It was written as a migration
instruction, at the point the T0 port consumed those files. The migration completed.

The clause outlived it. A reader now takes it as a standing parity obligation, and a difference
between an example and its template reads as a defect against the example.

## 2. What the measurement showed

**No served surface consumes the examples.** `design-system/designfs.go` embeds `templates/`,
`tokens/` and `fixtures/`. It names no `examples/` path, so the web app cannot read one. The
docs-site resolves `@ds/*` into `components/` and `tokens/` alone. A search for an import of any
examples path across the tree returns nothing.

One reader remains, and it is not a parity check. `design-system/console_tokens_test.go` reads
`examples/console/Settings.jsx` for the `--console-surface` invariant, which asserts one shared
token value and says nothing about a screen's composition. The Decision names that exception.

**The drift is systemic.** Twelve console screens carry it. Example commits and lines against
template commits and lines:

| Screen | Example | Template |
| --- | --- | --- |
| GraphView | 1 / 98 | 9 / 420 |
| Exposure | 2 / 59 | 5 / 108 |
| Coverage | 2 / 65 | 12 / 348 |
| Scope | 2 / 131 | 18 / 526 |
| Signals | 2 / 174 | 5 / 415 |

Every screen in that table was authored template-first against a SPEC. The example is the older
artefact in each pair, not the drifted one.

**Nothing enforces parity.**
[ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)
retired the byte-compare handoff that once held an example and its template together. What remains
is the `--console-surface` token invariant, which is a shared-value check and not a drift check.

**ADR-0110 was already two-thirds withdrawn.** Its `org switcher` clause fell to
[ADR-0181](./0181-the-deployment-is-single-tenant-so-no-organisation-is-modelled-and-the-shell-ships-a-static-chip.md).
Its `screenshots/` clause fell when `55aa367` deleted that tree. The verbatim-port clause is the
remainder.

## 3. What this changes

ADR-0110's Decision is withdrawn at the site that specifies it, under
[ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). Its `org
switcher` and `screenshots/` clauses were already struck, and the verbatim-port ruling is the
remainder, so nothing in that Decision survives this withdrawal. ADR-0110's Context and
Consequences stand as a record of what was decided in 2026-08. Four
documents restate that clause and move with it: `docs/agents/design-system.md`,
`.claude/skills/verge-asm-design/SKILL.md`, `design-system/README.md` and
`design-system/docs/DESIGN-NOTES.md`.

No example is deleted. Fourteen full-path citations across the ADR corpus, three SPECs and two
guides are existence-gated by the required `citations` check. Editing an example is free. Deleting
one breaks the build.

## 4. Consequences

A ticket that reports an example-to-template difference as a defect is now a non-event, and closes
without a repair. A session that edits a template incurs no obligation on the example beside it.

The cost is that the examples age. They record the T0 port and nothing refreshes them. A reader who
wants the shipped information architecture reads the SPEC and the template, which is what the
serving path already reads.

Reversal returns twelve open drift defects against files nothing consumes, and reinstates an
obligation no check can hold.
