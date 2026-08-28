# ADR-0116: the design package is normative for look AND functionality, so a missing datum is built, not re-skinned

- **Status:** Superseded (2026-08-28) — Design-system handoff workflow retired; the repo's served templates are the source of truth and may be edited in-repo. The parity gates (G1/G2) and the SPEC-CHANGE collision protocol this ADR established are withdrawn.
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
[`design-system/PARITY-CHART.md`](../../design-system/PARITY-CHART.md). The root cause was structural,
not a series of mistakes: a port-side judgment call split design authority in two, so the code became a
second source of truth that only its own comments knew about, and nobody upstream ever saw the
collisions to rule on them.

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
   The domain term still wins in *copy*; what this ADR removes is its authority over *structure*.

The scope of ADR-0024 is narrowed accordingly: it governs naming, not whether a specced region renders.
ADR-0110's verbatim-port mandate is affirmed and extended from IA to data shape.

**Every future domain–spec collision escalates, never decides port-side.**
[`design-system/SPEC-CHANGE.md`](../../design-system/SPEC-CHANGE.md) is adopted as the standing
protocol: a collision is filed there, the operator hands it to the design workspace, and design rules
one of three ways — build the datum, change the spec, or defer the region — each a recorded decision
rather than a buried judgment call. Its §Stop and escalate block is landed verbatim into `CLAUDE.md`
so it binds every session.

## Consequences

- **`CONTEXT.md`'s "a signal carries no severity" clause is superseded** and withdrawn at its site per
  [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md). A signal now
  carries a five-level severity (critical / high / medium / low / info) assigned per rule; the datum is
  built, not re-skinned (PARITY-CHART P0.1). The surviving true point — that a transition's *timing* is
  what makes a fact worth saying — stays; a `Message` still carries no severity
  ([ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md)).
- The parity backlog (map [#440](https://github.com/winniel123/verge-asm/issues/440)) builds the four
  missing model prerequisites — severity + per-instance signals, vs-last-batch deltas, trend series,
  and the ancillary reads — then restores the re-skinned regions across the shell and screens. Each
  ticket quotes its slice of `PARITY-CHART.md`.
- A reader who finds a spec region rendered as a placeholder in `cmd/web`, or a spec affordance missing,
  should treat it as drift against this ADR and file it, not preserve it. "Verbatim" claims from the
  original port are unverified until the closing 26-screen pass (PARITY-CHART P3).
- The cost is real schema and derivation work the port avoided. The ADR accepts it: a console that
  matches the design is the product, and one source of truth for look-and-functionality is worth the
  migrations.
