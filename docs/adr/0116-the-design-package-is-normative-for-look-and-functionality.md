# ADR-0116: the design package is normative for look AND functionality, so a missing datum is built, not re-skinned

- **Status:** Superseded (2026-08-28) — Design-system handoff workflow retired; the repo's served templates are the source of truth and may be edited in-repo. The parity gates (G1/G2) and the SPEC-CHANGE collision protocol this ADR established are withdrawn.
- **What survives the supersession** (added 2026-09-05, [#1410](https://github.com/winniel123/verge-asm/issues/1410), [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)): **the severity ruling in Consequences bullet 1 is live, and this ADR is still its only source.** A signal carries a five-level severity — critical / high / medium / low / info — assigned per rule, and that ruling withdraws `CONTEXT.md`'s older "a signal carries no severity" clause. `CONTEXT.md`'s `Signal` entry cites this ADR for exactly that rule. A reader who arrives from that citation must not read `Superseded` as retiring it. A `Message` still carries no severity ([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)).
- **What is unsettled:** whether Decision clauses 1–3 — build the datum, empty states for genuinely-empty data, no dropped or added affordances — survive the retirement of the workflow that carried them. This ADR does not say, and no later ADR rules it. `adr-gap` records [#1288](https://github.com/winniel123/verge-asm/issues/1288) and [#1300](https://github.com/winniel123/verge-asm/issues/1300) track the question. Do not read those clauses as settled in either direction.
- **Date:** 2026-08-24
- **Ticket:** [#441 P0.0 — Parity doctrine: ADR + SPEC-CHANGE protocol + CLAUDE.md stop-and-escalate](https://github.com/winniel123/verge-asm/issues/441)
- **Map:** [#440 Wayfinder: design-parity — make the console match the design package exactly](https://github.com/winniel123/verge-asm/issues/440)

## Context

The v3 port ([ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md))
took `design-system/examples/` as the console's IA spec and ported it. But where a spec region
rendered a datum the domain did not yet hold, the port ran a **local doctrine** to close the gap
port-side rather than escalate it:

> "the domain term wins and the visual convention gets re-skinned around it"; "fabricated mock data
> is re-skinned to honest current-state facts + empty-states."

That doctrine reads out of two places. From [ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)
the port took *the domain term is authoritative*, and extended it — beyond copy, where it belongs —
to **composition and data shape**, letting a domain gap justify redrawing the spec's region. From the
`reports.go` precedent and `docs/agents/design-system.md` it took *fabricated data is re-skinned to
honest current-state* and applied it to **structure**, replacing specced regions with placeholder
empty-states that point elsewhere.

Each individual call was well-reasoned and documented in a comment. The **sum** was a console that no
longer matches the design: severity dropped from four screens, spec regions replaced by placeholders,
the shell trimmed, trend series and deltas removed. The full drift audit is
~~`design-system/PARITY-CHART.md`~~, and the withdrawal below marks it gone. The root cause was structural,
not a series of mistakes: a port-side judgment call split design authority in two, so the code became a
second source of truth that only its own comments knew about, and nobody upstream ever saw the
collisions to rule on them.

> **The `PARITY-CHART.md` and `SPEC-CHANGE.md` citations in this ADR are WITHDRAWN at the sites
> that state them, 2026-08-28 by `55aa367` /
> [#1413](https://github.com/winniel123/verge-asm/issues/1413)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).**
> That commit retired the design-system handoff workflow and deleted both files.
> [ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md)
> states the rule that replaces the workflow. Every `PARITY-CHART Pn.n` and `SPEC-CHANGE #nn` token
> below resolves to nothing. The relative links to both files are unlinked here, because they led
> to a 404. The sentences stay as an accurate record of what this ADR read in 2026-08.

## Decision

**The design package (`design-system/`) is normative for look AND functionality.** The re-skin /
honest-current-state doctrine is **retired for composition and data shape**. Concretely:

1. **Where the domain lacks a datum the design renders** (severity, per-instance signals, deltas,
   trends, latency, instants), the fix is to **build the datum** — never to empty-state or reshape the
   spec's region.
2. **Empty states are for genuinely-empty data only**, drawn with the spec's own empty patterns. A
   spec region may not render as a placeholder that points elsewhere.
3. **No dropped affordances, no added ones.** What renders in the design preview is what ships. A
   wanted addition goes through the collision protocol first, never straight into the port.
4. **Vocabulary rules are unchanged and still binding** — signal / seed / channel / vantage /
   annotation, withdrawn never resolved ([ADR-0024](./0024-a-rules-domain-is-the-extension-of-its-name.md)).
   The domain term still wins in *copy*. What this ADR removes is its authority over *structure*.

The scope of ADR-0024 is narrowed accordingly: it governs naming, not whether a specced region renders.
ADR-0110's verbatim-port mandate is affirmed and extended from IA to data shape.

~~**Every future domain–spec collision escalates, never decides port-side.**
`design-system/SPEC-CHANGE.md` is adopted as the standing
protocol: a collision is filed there, the operator hands it to the design workspace, and design rules
one of three ways — build the datum, change the spec, or defer the region — each a recorded decision
rather than a buried judgment call. Its §Stop and escalate block is landed verbatim into `CLAUDE.md`
so it binds every session.~~

> **This clause is WITHDRAWN at the site that specifies it, 2026-08-28 by `55aa367` /
> [#1413](https://github.com/winniel123/verge-asm/issues/1413)
> ([ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)).** The
> status line withdraws the protocol by name. ADR-0058 rules that the unit is the sentence, so the
> clause carries its own marker. `design-system/SPEC-CHANGE.md` is deleted, and `CLAUDE.md` no
> longer carries the stop-and-escalate block. Nothing files a collision anywhere. A session that
> finds a domain–spec gap today edits `design-system/` in the repo
> ([ADR-0145](./0145-design-system-is-the-shared-home-and-source-of-truth-for-ui-assets-and-a-session-edits-it-in-the-repo.md),
> [`docs/agents/design-system.md`](../agents/design-system.md)).

## Consequences

- **`CONTEXT.md`'s "a signal carries no severity" clause is superseded** and withdrawn at its site per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). A signal now
  carries a five-level severity (critical / high / medium / low / info) assigned per rule. The datum is
  built, not re-skinned (~~PARITY-CHART P0.1~~ — a dead token, withdrawn in Context above). The
  surviving true point — that a transition's *timing* is what makes a fact worth saying — stays. A `Message` still carries no severity
  ([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)).
- The parity backlog (map [#440](https://github.com/winniel123/verge-asm/issues/440)) builds the four
  missing model prerequisites — severity + per-instance signals, vs-last-batch deltas, trend series,
  and the ancillary reads — then restores the re-skinned regions across the shell and screens. ~~Each
  ticket quotes its slice of `PARITY-CHART.md`.~~ That instruction is dead, withdrawn in Context
  above.
- A reader who finds a spec region rendered as a placeholder in `cmd/web`, or a spec affordance missing,
  should treat it as drift against this ADR and file it, not preserve it. "Verbatim" claims from the
  original port are unverified until the closing 26-screen pass (~~PARITY-CHART P3~~ — a dead
  token, withdrawn in Context above).
- The cost is real schema and derivation work the port avoided. The ADR accepts it: a console that
  matches the design is the product, and one source of truth for look-and-functionality is worth the
  migrations.
