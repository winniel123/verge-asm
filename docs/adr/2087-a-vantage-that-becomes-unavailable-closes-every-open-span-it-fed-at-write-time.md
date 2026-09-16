---
number: 2087
title: "A vantage that becomes unavailable closes every open span it fed, at write time"
slug: a-vantage-that-becomes-unavailable-closes-every-open-span-it-fed-at-write-time
date: 2026-09-16
status: accepted
source: grilling
ticket: 2087
proof: {ticket: 2137}
relations:
  - {kind: amends, adr: 108}
  - {kind: rests-on, adr: 80}
  - {kind: rests-on, adr: 6}
  - {kind: rests-on, adr: 41}
---

# ADR-2087: A vantage that becomes unavailable closes every open span it fed, at write time

## Decision

> **A vantage that becomes unavailable closes every open span it fed, at write time**, and opens a
> `Gap` carrying the cause `vantage-unavailable` behind each. The transition is the event, not the
> column value, so a resolver-only vantage holding no value still transitions. The closure names no
> facet, so one added later needs no query edit. No composition read gains an availability
> predicate, and `ListVantagesForDispatch` keeps every row.
>
> A reader asks why a class-scoped composition trusts a prober that stopped answering. ADR-0080
> decision rule 1 says it must not, and no read implemented that predicate.
>
> Rejected: a predicate on each composition read. `applyAvailability` clears a vantage only from a
> completed resolution-walk, so filtering the dispatch query would strand it unavailable forever.
>
> Reversal returns a dead prober's open span to the fold, pinning its class to `reached`.

## 1. Context

[ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)
decision rule 1 says a class-scoped composition reads the available vantages of that class alone, and
that an empty set of available vantages is not-evaluable. Its cross-class limb adds that a class with
no available vantage leaves the comparison unmade.

No read implemented that predicate. The rule was written and never built, and nothing failed loudly,
because a dead prober's last reading looks exactly like a live prober's current one.

## 2. Two paths, and neither implemented the predicate

Both were measured before this decision.

| Path | Read | Hole |
| --- | --- | --- |
| Resolution | `ListVantagesForDispatch` | Applies no predicate and does not select the `availability` column, so an unavailable vantage still names its class. |
| Resolution | `ListNameResolutionsByClass` | Carries no predicate, so the value half is unfiltered too. |
| Reachability | `ListServiceReachabilitySpansByClass` | The four reads have the same hole. One dead prober pins a class to `reached` under the existential fold. |

The reachability case is the sharper one. An existential fold needs a single `reached` to conclude,
so one prober that stopped answering holds its class at `reached` for as long as its span stays open,
which is forever.

## 3. Why the write side

> **Amended** by [ADR-2163: A facet is reached by the aperture of the corpus it composes from](./2163-a-facet-is-reached-by-the-aperture-of-the-corpus-it-composes-from.md), 2026-09-16. <!-- adr-marker amends 2163 -->

A `Gap` is what the corpus already has for "we could not say". Closing the span at the write makes
every reader correct at once, including readers not yet written, and it needs no reader to remember
a predicate.

It also keeps the exclusion in one place. A read-side rule is a rule every future read must re-learn,
and the record shows that cost: ADR-0080 wrote the predicate and five reads did not carry it.

[ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md) bounds the
shape. The write closes a span and opens a `Gap`. It deletes nothing and rewrites no value.

## 4. Every facet, not two

The first cut closed `reachability` and `resolution`. That left the same dead vantage's `dns-record`
spans open, which is the same defect one facet over.

So the closure names no facet at all. `certificate`, `http-identity`, `tls-acceptance` and any facet
added later are covered without a query edit and without amending this ADR. A facet list in the query
would be a second place to forget.

The `Gap` value's outcome spelling is per-facet, because each facet's own writer owns it. That is a
spelling and not a scope, and the two must not be confused.

## 5. The rejected alternative

**Filter each composition read on `vantage.availability`.**

It is rejected on a measured hazard, not on taste. `applyAvailability` restores a vantage to
`available` only from a completed resolution-walk batch. Filtering the dispatch query itself would
therefore strand an unavailable vantage as unavailable forever: it would never be dispatched, so it
could never complete the walk that clears it. `ListVantagesForDispatch` has five callers, and the
dispatch job builders in `internal/queue` must keep every row.

A narrower read-side filter, on the composition reads alone, escapes the stranding but not the
second objection.
[ADR-0006](./0006-subjects-leave-by-measurement.md) refuses a conclusion drawn from the survivor.
Filtering the cross-class denominator concludes from the class that still answers, so an `Exposure`
that would need the missing class must be absent rather than quietly computed.

## 6. What this ADR does not rule

**The recovery side.** Closing a `Gap` when a vantage recovers is bounded by what the recovering
batch re-measured, because a resolver signal says nothing about connect health, exactly as a port
probe says nothing about resolver health. The residual — a timeline that receives no further batch of
its own facet — is recorded in #2189 and is not decided here.

**The resolution read.** `ListNameResolutionsByClass` selects from the observation corpus rather than
the span corpus, so closing spans changes no row it returns. #2163 carries that.

**Nullability.** `vantage.availability` is nullable, and the shipped `local` vantage is resolver-only
and holds no value. This ADR rules that the transition is the event, so a NULL is never read as
`unavailable`. It rules nothing else about the column.

## 7. Consequences

A dead prober stops voting. `Coverage` gains a `Gap` per closed timeline rather than a silent
absence, and a class whose only vantage went dark reads not-evaluable rather than a vacuous
`not-reached`.

A standing comment on `runningVantageClasses` asserted the opposite: that an unavailable vantage
still names its class and the gap reads not-evaluable. This ADR overrules that comment, and it is
deleted rather than left to contradict the rule.

`CONTEXT.md` stated the `Gap` rule for `Reach` alone and said nothing about `resolution`. Under this
ADR it states both, and names no facet.

Reversal returns a dead prober's open span to the fold, pinning its class to `reached`.

Nothing executes the statement. No Postgres runs on the development machine, and CI's `compose` job
backfills zero rows on a fresh database, so a green `compose` job is not evidence here. #2166 carries
that gap.

## 8. How ADR-0108 is amended

[ADR-0108](./0108-a-batch-whose-instrument-could-not-reach-its-position-covers-nothing-and-the-failure-is-the-vantages.md)
said an unavailable vantage is excluded at read time, in `cmd/web/exposure.go`. That file never read
the `availability` column, and under this ADR no read ever will.

ADR-0108 numbers no heading, so this ADR declares `{kind: amends, adr: 108}` with no `clause`, and
the tool writes the marker under its H1. ADR-0108's own prose is not hand-edited.

Nothing else in ADR-0108 moves. Its rule that a batch whose instrument could not reach its position
covers nothing, and that the failure is the vantage's, is what this ADR builds on.

## 9. Proof

`{ticket: 2137}`. #2137 is open and names this rule: close a vantage's open spans and open a `Gap`
when it becomes unavailable.
