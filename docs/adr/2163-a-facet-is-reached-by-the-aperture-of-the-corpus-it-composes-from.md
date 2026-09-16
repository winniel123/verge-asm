---
number: 2163
title: "A facet is reached by the aperture of the corpus it composes from"
slug: a-facet-is-reached-by-the-aperture-of-the-corpus-it-composes-from
date: 2026-09-16
status: accepted
source: grilling
ticket: 2163
proof: {test: "internal/vantage/availability_test.go::TestEachFacetIsReachedByTheApertureOfItsOwnCorpus"}
relations:
  - {kind: amends, adr: 2087, clause: "3"}
  - {kind: rests-on, adr: 80}
  - {kind: rests-on, adr: 41}
---

# ADR-2163: A facet is reached by the aperture of the corpus it composes from

## Decision

> **A facet is reached by the aperture of the corpus it composes from.** A write-time closure
> reaches the corpus it writes, no other. Reachability composes from `span`, so ADR-2087's
> closure reaches it. Resolution composes from `observation`, and its aperture is the cadence
> floor: at the shipped daily `dns` cadence a dead vantage's class is stale for at most 48
> hours, then reads not-evaluable. That floor is intended, not a hole.
>
> A reader asks why ADR-2087 §3 promises every reader correctness while §6 admits a read it
> cannot reach. §3 names no corpus, so the next observation read is written assuming cover.
>
> Rejected: a second writer edge on `observation`. ADR-0041 rules that corpus, and writing over
> it costs more than a bounded, self-healing window.
>
> Reversal unprices the window and returns §3's unqualified promise.

## 1. Context

[ADR-2087](./2087-a-vantage-that-becomes-unavailable-closes-every-open-span-it-fed-at-write-time.md)
rules that a vantage which becomes unavailable closes every open span it fed, at write time, and that
no composition read gains an availability predicate. §3 gives the warrant: closing at the write
"makes every reader correct at once, including readers not yet written".

§6 of the same ADR then records a residual. `ListNameResolutionsByClass` selects from the observation
corpus, so closing spans changes no row it returns. #2163 was opened to carry it.

The two sentences read as a contradiction because §3 names no corpus. It says *every reader*, and the
reach it actually has is *every reader of `span`*. An author who acts on §3 alone writes an
observation-corpus read and assumes the closure covers it.

This ADR states the missing qualifier, and prices what the qualifier costs.

## 2. Which corpus each facet composes from

`MarkVantageUnavailable` in `db/queries/vantages.sql` is one statement. It moves
`vantage.availability`, closes every open `span` at that vantage, and opens a `Gap` behind each. It
touches no row in `observation`.

| Facet | Composition read | Corpus | Instrument that bounds its staleness |
| --- | --- | --- | --- |
| `reachability` | `ListServiceReachabilitySpansByClass` and its three siblings | `span` | the write-time closure of ADR-2087 |
| `resolution` | `ListNameResolutionsByClass` | `observation` | the cadence floor, `floor_cadences * tightest_cadence` |
| `dns-record` | `ListNameDNSRecords` | `observation` | the cadence floor |
| any facet added later | — | whichever it selects `FROM` | that corpus's own instrument |

The last row is the rule, and the first three are its instances. A facet added later inherits the
aperture of the table its read names, and needs no edit here to do so.

Both instruments are already correct on their own terms. A span is a settled interval, so the only
way to stop it voting is to close it. An observation is a dated reading, so the only way to stop it
voting is to let it age out. Neither is a substitute for the other.

## 3. The window, priced

`retention.FloorCadences` is `2`. The `dns` scan ships enabled at `cadence_seconds` 86400, and it is
the only scan that dispatches a resolution walk, so `tightest_cadence` for a `resolution` row is
86400 seconds.

`2 x 86400 = 172800 seconds`. **48 hours.**

That is the worst case, and it is reached when a vantage goes unavailable immediately after a fresh
reading. The observed window is the remainder of the floor, between zero and 48 hours. After it the
observation falls outside the floor, the class holds no value, and `composeResolution` reads it as
not-evaluable, which is what ADR-0080 decision rule 1 asks for.

The figure moves with the operator's cadence dial. The shape does not: the window is always
`FloorCadences` times the tightest enabled cadence that covers the row.

Three properties are what make 48 hours acceptable. It is bounded. It is self-healing, with no
operator act and no repair job. And it converges on the right answer rather than a wrong one, because
the terminal state is not-evaluable.

## 4. The rejected alternative

**A second writer edge, closing or superseding the observations of an unavailable vantage.**

Rejected on cost and on shape.
[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) rules the
observation corpus by what may still read it. An observation is a dated reading of what a position
saw, and a vantage going dark later does not unsee it. Writing over the corpus to express "this is no
longer current" restates in data what the cadence floor already computes, and it puts the currency
rule in two places that must then agree.

The read-side predicate is rejected too, and ADR-2087 §5 already carries that rejection. Nothing here
reopens it.

## 5. What this ADR does not rule

**The reachability path.** #2137 closed it at the source and the span read carries it. This ADR
describes that path but changes nothing about it.

**The recovery side.** `MarkVantageAvailable` retires the outage `Gap` only on the facets the
recovering batch re-read. #2189 carries the residual, and #2250 records the flap it produces.

**Which facets a batch outcome gaps.** ADR-2087 §4 rules the closure names no facet, and #2165
carries an unratified proposal to keep it that way. This ADR describes the behaviour under §4 and
revisits neither.

**The standing of the Postgres harness.** `internal/dbtest` executes `MarkVantageUnavailable` against
a real PostgreSQL, and CI's `query-harness` job runs it and fails when a case skips. That job is not
a required check, so a red harness holds no merge. #2166 carries the standing.

## 6. How ADR-2087 §3 is amended

ADR-2087 numbers its headings, and §3 is *Why the write side*. So this ADR declares
`{kind: amends, adr: 2087, clause: "3"}`, and the tool writes the marker under that heading.
ADR-2087's own prose is not hand-edited.

§3's warrant stands. Closing at the write is still the right instrument, it still keeps the exclusion
in one place, and it still needs no reader to remember a predicate. One limit joins it: *every
reader* means every reader of the corpus the closure writes.

This ADR also carries the ADR half of #2202, which asked for exactly that qualifier and was ruled to
ride this ticket rather than earn a second ADR on one sentence. #2202's CONTEXT.md half is separate
and lands on its own.

## 7. Consequences

An author who reads §3 now reads which corpus it reaches, and an author adding a facet reads which
instrument bounds it, in one table.

The 48-hour window is priced. It was unpriced before, which is the whole reason #2163 could not be
closed by reading the code.

The issue text of #2087 states the consequence as *"the corpus reads correctly with no read-side
predicate"*. That is true of `span` and not of `observation`, and the narrowed wording is posted
there.

No query changes. `ListNameResolutionsByClass` is correct as written, so
`TestNoCompositionReadGainedAnAvailabilityPredicate` stays honest and is not amended.

Reversal unprices the window and returns §3's unqualified promise, leaving the next
observation-corpus read to rediscover the gap.

## 8. Proof

`{test: "internal/vantage/availability_test.go::TestEachFacetIsReachedByTheApertureOfItsOwnCorpus"}`.
The test asserts the table of §2 on each query it names: the closure writes `span` and never
`observation`, all four reachability reads select `FROM span` and never `FROM observation`, under
`closed_at IS NULL`, and the resolution read selects `FROM observation` and never `FROM span`, under
`floor_cadences * tightest_cadence`. Each corpus assertion is two-sided, so a read that grew a join
onto the other corpus fails rather than passing on the substring it kept.

It also multiplies `retention.FloorCadences` by the `dns` cadence the last migration to write it
sets, so the 48 hours of §3 is checked rather than remembered, and a re-priced dial fails here
before the figure goes stale.
