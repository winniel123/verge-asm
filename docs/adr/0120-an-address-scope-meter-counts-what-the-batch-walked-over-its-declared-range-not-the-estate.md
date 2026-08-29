# ADR-0120: an address-scope aperture meter counts what the batch walked over its declared range, not the estate

- **Status:** Accepted
- **Date:** 2026-08-25
- **Ticket:** [#551 Coverage exact-parity conversion — data + fixtures](https://github.com/winniel123/verge-asm/issues/551)
- **Map:** [#545 batch 1 — SignIn + Setup + Coverage (v3.7.0)](https://github.com/winniel123/verge-asm/issues/545)
- **Refines:** [ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)

## Context

The v4 exact-parity conversion of the Coverage screen (WORKFLOW v4, package v3.7.0)
lands a frozen `coverage.tmpl` whose aperture meter has **two shapes**, where the ported
V3 screen had one. The design ruling recorded in
[`design-system/SPEC-CHANGE.md` #19c](../../design-system/SPEC-CHANGE.md) (owner, 2026-08-25)
is:

> an ADDRESS scope's meter renders `counted / total` — denominator = the enumerable
> addresses of the declared range (a /24's usable size), counted = subjects the batch
> walked; NAME scopes stay census. Record as an ADR note refining ADR-0095
> (estate-proportion stays forbidden; a declared range is not the estate).

[ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)
and its lineage (#44 decision 7, ADR-0044, ADR-0072) bar the aperture statement from
carrying **a count or proportion of the operator's estate** — "counts **our** lists and
rules; carries **no** count of what is unmeasured". A denominator that is *the estate* is
[#28](https://github.com/winniel123/verge-asm/issues/28)'s refused completeness score. The
question this note settles is whether a `counted / total` **address-scope** meter collides
with that bar.

## Decision

**It does not, and the reason is that a declared address range is not the estate.**

| Concern | Ruling |
| --- | --- |
| What an ADDRESS scope's meter renders | `counted / total` with a proportional fill (`Pct`, precomputed 0–100) |
| The denominator | **The enumerable addresses of the declared range** — a finite, operator-declared set (a /24's usable size), fully known before any measurement. Not the estate, not a completeness target |
| The numerator | **The subjects the batch walked** within that range this cycle |
| A NAME scope's meter | **Unchanged — a census.** A name scope enumerates nothing on its own (its addresses arrive by resolution), so it has no denominator and renders the striped census bar |
| The estate-proportion bar | **Untouched.** ADR-0095 / #44 decision 7 forbid a proportion *of the estate*; a proportion *of a declared range the operator themselves enumerated* is a census **of that range**, and the range is a closed set we own, not the open estate we do not |
| Why this is not #28 in disguise | #28's score divides by *the estate's true size*, which is unknowable. This divides by *a declared range's enumerable size*, which is arithmetic over the operator's own declaration — the same character as `5 of 38 sensitive pairs` (a set-arithmetic figure over our own list) that ADR-0095 explicitly permits |

## Consequences

- **ADR-0095's estate-proportion prohibition is confirmed, not weakened.** It bars a
  denominator that is the estate. A declared address range is a bounded set the operator
  enumerated. `counted / total` over it is a census of that range. The two never fuse: a
  name scope, whose addresses are *not* enumerable in advance, keeps the census bar with no
  denominator, exactly as before.
- **The denominator is a constant of the declaration**, in the spirit of ADR-0044's
  constancy premise: it is the enumerable size of the declared range, fixed the moment the
  scope is declared, and moves only when the operator edits the scope — never with the
  weather, never with a probe cycle.
- **Realization today is the fixture-seeded instance.** The `counted / total` figures the
  design mock and the pixel golden depict (`203.0.113.0/24` → `198 / 214 subjects`, with the
  "16 skipped: excluded subtree + 3 unresolvable names" breakdown) are the design's curated
  corpus, served byte-for-byte from the pinned Coverage fixture under a `VERGE_DEV` build
  (`cmd/web/devfixtures.go`, drift-gated by `TestCoverageFixtureMatchesPackage`). This is
  the exact-parity target G2 verifies.
- **A live estate renders the honest census, pending a first-class numerator.** The
  denominator (enumerable addresses of a declared range) is already a first-class datum.
  The numerator — *subjects the batch walked within a declared range* — is **not** a
  first-class read yet, so `cmd/web/cold.go`'s live `coveragePage` renders the address meter
  as a census rather than fabricate the numerator (which
  [`SPEC-CHANGE.md`](../../design-system/SPEC-CHANGE.md) forbids). Wiring the live
  `counted / total` form is carried as a successor: it needs a per-scope walked-subject
  count, not a spec change — this ruling is the spec.
- **No coverage-class member, message cause, or aperture input is minted.** This is a
  rendering refinement to one existing aperture meter. ADR-0095's ten-member class, the
  seven aperture inputs, and the four/five absence registers are untouched.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Keep every meter a census (no `counted / total` at all) | Contradicts the design ruling (#19c). The operator cannot tell how much of a *declared, enumerable* range the batch actually walked — a real currency question the census bar cannot answer |
| Make the denominator the estate size, so both scopes get a proportion | The exact figure ADR-0095, ADR-0044 and #28 refuse: a proportion of the estate is a completeness score wearing a coverage costume, and the estate size is unknowable |
| Give NAME scopes a `counted / total` too | A name scope enumerates nothing in advance — it has no enumerable denominator — so any denominator would be invented. The census bar is the honest shape and stays |
| Fabricate the live numerator (e.g. reuse an unrelated subject count) | Forbidden by SPEC-CHANGE: an approximated datum shipped as fact is the drift this protocol exists to prevent. The live meter stays an honest census until the walked-subject count is a real read |
