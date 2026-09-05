# ADR-0147: "assets watched" is the distinct-subject count over the open-span corpus

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1288 ADR gaps: internal/drift](https://github.com/winniel123/verge-asm/issues/1288)
- **Rests on:** [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md) (one corpus carries more than one projection, and the open-span read is not routed through the live-tier observation gate) and [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) (the span corpus is never compacted, so a past population is rebuilt rather than stored)
- **Constrained by:** [ADR-0072](./0072-absence-is-a-property-of-a-cell-and-withdrawn-is-the-only-population.md) (a listing states no denominator), [ADR-0082](./0082-a-withdrawn-subjects-timelines-close-and-the-withdrawn-period-is-on-no-timeline.md) (a withdrawal closes the timelines), [ADR-0102](./0102-a-subjects-row-is-the-base-a-census-member-row-is-its-explicit-modifier.md) (a denominator beside a subject count is a claim of estate completeness)
- **Amends:** [ADR-0110](./0110-the-design-system-examples-are-the-consoles-ia-spec-ported-verbatim.md), whose Decision carries the withdrawn-never-resolved vocabulary rule, with the KPI consequence that rule forces

## Context

The dashboard stat band draws a tile labelled **Assets watched**. It draws one label over **two
different reads**, and they can disagree.

**The value is a subject listing.** `cmd/web/auth.go:688` computes `assetsWatched := names + services`.
`names` is `len(rows)` from `ListCurrentNameSubjects` (`cmd/web/auth.go:556`) and `services` is
`len(rows)` from `ListCurrentServiceSubjects` (`cmd/web/auth.go:566`). Both queries live in
`db/queries/subjects.sql`. Both read the **observation** table through a freshness gate —
`as_of - observed_at <= floor_cadences * tightest_cadence` — and the name query additionally drops any
subject whose latest resolution outcome is `NameError` or `Shadowed`.

**The delta is an open-span fold.** `cmd/web/deltas.go:93` computes
`drift.CountDelta(assets, prevAt, drift.DistinctSubjects)` over the rows of `ListSpansOpenSince`,
pre-filtered to `subject_kind` `name` or `service`. `CountDelta` folds twice over one row set —
`DistinctSubjects(CurrentlyOpen(all))` and `DistinctSubjects(OpenAt(all, prevAt))` — and the tile
renders the difference (`cmd/web/auth.go:697`).

So the tile's **number** comes from the observation tier and the tile's **arrow** comes from the span
corpus. Three mechanisms drive them apart, and none is exotic:

1. **The freshness gate.** A subject whose scan cadence slips past `floor_cadences` drops out of the
   listing while its span stays open. The value falls. The delta does not move.
2. **The membership filter.** A `Name` whose latest resolution is `NameError` or `Shadowed` is filtered
   out of the listing. ADR-0105 §3 states that the same subject **appears** in an open-span read,
   carrying that outcome as its current value. The two reads disagree by construction.
3. **The facet count.** The listing takes one row per `subject_key` off a single facet — `resolution`
   for names, `reachability` for services. `DistinctSubjects` folds every open facet timeline a subject
   holds down to one `(subject_kind, subject_key)`. They agree today only because the listing already
   picked one facet, and nothing holds that coincidence in place.

An operator can therefore read `1,284 · +12` where the twelve were counted against a base that is not
1,284. The tile is the first number on the first screen, so a number and an arrow that answer different
questions is not a cosmetic defect.

**Two corrections to #1288's record.** The Reports page carries **no** assets-watched tile —
`cmd/web/reports_test.go:109` asserts it must not, and the card there is "New assets discovered", a
`drift.DiscoveryCount` over first appearances. And the heatmap ramp has **one** consuming surface, not
two: `cmd/web/reports_export.go:53` calls `bucketScanActivity` alone and never reaches
`drift.HeatLevels`.

## Decision

> **"Assets watched" is the count of distinct `Name` and `Service` subjects holding at least one open
> span at the instant of the read. It is `DistinctSubjects(CurrentlyOpen(spans))` over the
> name/service-filtered open-span corpus, and the tile's value and its delta are two folds of that one
> read. The subject listing is not the definition.**

### 1. The population is an open span, not a fresh observation

A subject is watched when it holds an open span. That is the same fact estate membership rests on
(ADR-0105 §3), and it is read off the never-compacted span corpus (ADR-0041) rather than the
observation tier, so no `as_of` bound applies and a settled current state is never hidden by a cadence
gate.

This matters most at the edges the listing read handles differently. A subject whose scan slipped a
cadence is still watched — verge is still watching it, and the slip is a coverage fact the coverage
meters report, not a shrinking estate. A `Name` resolving to `NameError` is still watched: ADR-0082
closes a timeline on **withdrawal**, and a `NameError` is a value, not a departure.

### 2. A subject counts once, whatever its facet timelines

`DistinctSubjects` keys on `(subject_kind, subject_key)`. A `Name` holding open `resolution`,
`dns-record`, `certificate` and `http-identity` timelines is **one** watched asset. Enabling a facet on
a scan therefore never moves this number, which is the property that makes the tile readable across a
configuration change.

### 3. The value and the delta are folds of one read

The tile's value is the **current** arm of the fold it already computes. `CountDelta` returns
`Delta{Current, Previous}`, and `Current` is exactly the number this ADR defines. The value and the
arrow are then arithmetically consistent — `Change() = Current - Previous` — rather than consistent by
luck.

This is ADR-0105's rule applied a third time. Same rows, another projection: no new table, no new
observation, no new `Derivation` leaf, no new value in any facet's space.

### 4. The count carries no denominator

The tile states a number and never a share. ADR-0072 refuses an estate denominator, and ADR-0102 names
a denominator beside a subject count as a claim of estate completeness the product refuses everywhere
else. So there is no *"1,284 of N"*, no coverage percentage on this tile, and no partition of it.

The tile's caption stays what it is — the operator's own declared scopes ("8 domains · 3 ranges"). That
is an **act count**, not a denominator: it says what the operator asked verge to watch, and the tile
says what verge holds open. The two are not a numerator and a total, and the tile never presents them
as one.

### 5. Summing `Name` and `Service` is bounded to this tile

`Name + Service` is a sum across two subject kinds, and ADR-0072 §2 forbids exactly such a sum in a
`Subjects` **listing header**, where the two lists are `Name` and `Address` and summing them counts one
membership fact twice.

This tile is not that header. It sums two kinds that are never peers in one list, it is a single
headline count rather than a partition of a listing, and it states no denominator. The sum is therefore
admissible **here and nowhere else**: no listing header, no census-member row, and no per-kind
breakdown derived from it.

## Consequences

- **This ADR changes no production Go code.** It records the definition. Aligning
  `cmd/web/auth.go:688` onto the fold at `cmd/web/deltas.go:93` is a follow-up ticket.
- **The tile's number will change on the estates where the two reads disagree.** A stale-cadence or
  `NameError` subject the listing dropped will be counted. That is the point: it is watched, and the
  delta already counted it.
- **`emptyEstate` keeps the listing read.** `cmd/web/auth.go:648` gates the first-run checklist on
  `names == 0 && services == 0`. That question is *has any measurement landed yet*, which the
  observation tier answers and the span corpus does not. It is a different question and it keeps its own
  read.
- **The tile degrades with the delta, not separately.** When `dashboardDeltas` returns `statDeltas{}` —
  no previous batch instant, or a failed span read — the tile has no number to show and shows none, in
  place of today's split behaviour where the value renders and the arrow does not.
- **`CONTEXT.md`'s `Asset` entry gains one sentence.** The term stays refused as a modelled thing and
  stays admitted as an interface collective noun. What it lacked was the rule that fixes what the
  interface count means when it uses the noun.
- **ADR-0110 gains the KPI consequence** beside its withdrawn-never-resolved vocabulary rule. The rule
  was stated as vocabulary, and its measurement consequence lived only in a code comment.
- **`DistinctSubjects` is the definition's one home.** A second caller counting watched assets calls
  that fold. A hand-rolled `len()` over a subject listing is a regression under this ADR.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **The subject listing is the definition** — fold `ListCurrentNameSubjects` + `ListCurrentServiceSubjects` for the delta too | It defines the watched estate by observation **freshness**, so a scan that slips a cadence shrinks the estate rather than reporting a coverage gap. It re-implements the `NameError`/`Shadowed` membership filter at the counting layer, which ADR-0105 §3 removed by construction. And it cannot produce a previous count at all: the observation tier has no *as of the previous batch* read, so the delta would need a stored counter — a new thing to measure, which ADR-0105 §1 refuses |
| **Keep both reads and reconcile at render** — show the listing value, clamp the delta to it | It preserves the defect and hides it. The clamp discards real movement, and the tile reports a change smaller than the one that happened |
| **Count open spans rather than distinct subjects** (`CountSpans`) | The number moves whenever a facet is enabled on a scan, with no change in the estate. `CountSpans` exists for populations where a span **is** the unit; a watched asset is a subject |
| **Add `Address` and `Endpoint` to the population** | An `Endpoint` is reached through a root and is never a peer in a listing (ADR-0072 §2), and an enumerated address scope admits every address in a declared CIDR, so one `/22` would put 1,024 addresses into a headline count meant to read as *things we watch* |
| **State the count against an estate denominator** | ADR-0072 refuses it, and ADR-0102 names it as the most dangerous form of #28's hazard: a false assertion about the shape of the whole attack surface, on the screen the operator opens first |
| **Rename the tile to "Subjects watched"** and drop the collective noun | It resolves the ambiguity by retreating from the interface vocabulary `CONTEXT.md` explicitly admits. The defect is two reads under one label, not the label |
