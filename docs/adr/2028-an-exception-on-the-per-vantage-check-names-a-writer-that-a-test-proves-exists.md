---
number: 2028
title: "An exception on the per-vantage CHECK names a writer that a test proves exists"
slug: an-exception-on-the-per-vantage-check-names-a-writer-that-a-test-proves-exists
date: 2026-09-15
status: accepted
source: fix
ticket: 2028
proof: {test: "internal/measure/edgefanout/leaf_test.go::TestEmitDeclaresNoFacet"}
relations:
  - {kind: amends, adr: 1985, clause: "4"}
  - {kind: rests-on, adr: 217}
  - {kind: rests-on, adr: 129}
---

# ADR-2028: An exception on the per-vantage CHECK names a writer that a test proves exists

## Decision

**An exception arm on the per-vantage `CHECK` names a writer in this tree, and the comment
at the arm names the test that proves that writer exists. An arm that names neither is
dropped.**

Three of the four arms name neither. `OR facet = 'certificate'` leaves `span` and
`observation`, and `OR source = 'zone'` leaves `span`. The `observation` zone arm stays,
and its comment gains its test.

A reader asks why, because ADR-1985 §4 read the certificate arm off the edge fan-out. That
fan-out declares no facet. The arm therefore admits one row only: a `certificate` line an
untrusted prober injects onto a vantage-less job, which is #1985 reopened on that facet.

Rejected: narrowing the arm by source, which keeps an exception no writer needs. Rejected:
closing the recording-side gate instead, which leaves the constraint overstated.

Reversal re-admits that row on both tables.

## 1. Context

[ADR-1985](./1985-a-timeline-row-on-a-per-vantage-facet-is-refused-by-the-database-without-a-vantage.md)
rules that a timeline row on a per-vantage facet cannot be stored without a vantage.
`db/migrations/26200_per_vantage_facet_needs_vantage.sql#span_per_vantage_facet_needs_vantage`
implements it twice, once on `span` and once on `observation`. Both predicates read the same:

```sql
vantage_id IS NOT NULL
OR source = 'zone'
OR facet = 'certificate'
```

ADR-1985 §4 argues the shape. The predicate is a deny-list, so a facet added later is constrained on
the day it is added, and a genuinely vantage-less facet must argue its way onto an exception line.
The argument the section accepts is a producer: `certificate` is said to be *"mixed inside one
source, because `connect-outcome` sets the vantage and the edge fan-out nulls it"*.

The code review on [#2025](https://github.com/winniel123/verge-asm/pull/2025) read that sentence
against the tree and could not find the fan-out's certificate. It recorded the finding and changed
nothing, because narrowing a merged constraint is a decision an ADR carries.

This ADR holds that the argument a line makes has to be checkable. Four arms exist. One writer does.

## 2. The certificate arm's writer does not exist

ADR-1985 §4 cites [ADR-0129](./0129-a-shared-foreign-edge-is-measured-by-fan-out-not-read-from-a-list.md)
and [#954](https://github.com/winniel123/verge-asm/issues/954). ADR-0129 rules that a shared foreign
edge is measured by fan-out, and the enqueue at
`internal/queue/edgefanout.go#enqueueEdgeFanoutJob` does write a NULL `vantage_id`, with a comment
that says a default certificate is not per-vantage. That intent is real. It never reached a
facet-bearing writer.

- The leaf declares no facet. `internal/measure/edgefanout/emit.go#Emit` returns an observation whose
  `Facet` is empty, and `internal/measure/edgefanout/leaf_test.go#TestEmitDeclaresNoFacet` asserts
  it.
- The recorder refuses one. `internal/queue/edgefanout.go#toEdgeFanoutRows` drops any line that
  carries a facet, and the rows it keeps land in `edge_fanout_observation`. That table holds no
  vantage and no facet column.
- The only certificate emitter is per-vantage.
  `internal/measure/connectoutcome/certificate.go#EmitCertificate` stamps `scope.Vantage`, and it
  runs on a `connect-outcome` job. `internal/scan/cold.go#BuildColdJobs` and
  `internal/scan/hot.go#BuildHotJobs` both stamp the loop's vantage id.
- The fixture seeder cannot reach the arm either.
  `cmd/web/seedfixtures.go#seedInventoryFixtures` now resolves a vantage row per fixture span and
  fails the seed where the name is unknown, so every row it writes carries a vantage.

So no honest producer needs the arm. The arm is not therefore inert, which §3 takes up.

## 3. What the arm does admit

`internal/queue/worker.go#Worker.process` routes `zone`, `ct` and `ct-tail` to their own completions.
Every other kind reaches `internal/queue/worker.go#Worker.complete`, and the edge fan-out is the one
kind that arrives there with a NULL `vantage_id`. Inside that completion two writers read the job's
vantage and the prober's facet together:

- `internal/queue/pure.go#toObservationParams` writes one `observation` row per facet-bearing line,
  with the job's `vantage_id` and `internal/queue/pure.go#SourceFor` as the source.
- `internal/queue/spanfold.go#foldOne` opens one `span` per facet-bearing line, on the same vantage.

The recording-side gate stands in front of both.
[ADR-0217](./0217-the-recording-side-scope-gate-gates-a-denoted-dimension-alone-and-only-the-edge-fanout-facet-less-arm-fails-closed.md)
rules that the gate gates a denoted dimension alone, and
`internal/queue/scopegate.go#authorizedScope.admits` gates a `certificate` line on its subject
address. An edge fan-out scope denotes addresses, and the subject of an injected certificate line can
name one of them. The gate admits the line.

A compromised or defective prober can therefore produce, on an edge fan-out job, a `certificate` row
on both tables with a NULL vantage. `OR facet = 'certificate'` is the only reason the database
stores it. That row survives `db/queries/span.sql#ListAllOpenSpans` and loses its legs to
`db/queries/signals.sql#ListServiceReachabilitySpansByClass`, which is the false aperture claim
ADR-1985 exists to make unrepresentable, on the one facet a lying prober can reach.

The finding is therefore worse than a dead carve-out. The arm admits nothing an honest writer emits,
and exactly the row an untrusted one does.

## 4. The `span` copy of the zone arm is dead by construction

`OR source = 'zone'` on `observation` is load-bearing.
`internal/queue/zone.go#toZoneObservationParams` writes `dns-record` rows with `scan.ZoneSource` and
no vantage, and `internal/queue/zone_test.go#TestToZoneObservationParamsStampsSupplyInstantAndZoneSource`
proves it. Without the arm the zone reader's rows are refused.

The `span` copy admits nothing, and no prober can change that. `internal/queue/zone.go#Worker.completeZone`
writes observations and opens no span, which
[ADR-0027](./0027-a-source-may-admit-without-observing.md) already rules: a source may admit without
observing. Every span in the tree comes from `internal/queue/spanfold.go#foldOne` or from the fixture
seeder, and both set `source` from `internal/queue/pure.go#SourceFor`. That switch returns `prober`
or `resolver` and nothing else. A prober names its facet, never its source, so the value `zone` is
unreachable on `span` by any input.

This is a stronger absence than §2's. The certificate arm has no writer today. The span zone arm has
no writer that the code admits the shape of.

## 5. Why an arm names a test

ADR-1985 §4 asks a new facet to argue its way onto the exception line. It does not say what a
sufficient argument is, and the certificate arm shows the cost: a plausible sentence about a producer,
citing a real ADR and a real issue, that no writer ever exhibited. The sentence stayed true-looking
for as long as nobody re-read it against the tree.

A named test is the smallest thing that cannot rot quietly. A rename or a deletion of the writer
breaks the test, and the `test` proof rule of `docs/spec/adr-governance.md` §6 already teaches the
repository that shape. The arm's comment names the test, so the next reader of the migration reaches
the evidence in one hop rather than reconstructing §2 by hand.

No CI check enforces this. The rule binds an author and a reviewer, and a broken arm surfaces as a
failing test in the package that owns the writer, never as a red constraint. That limit is accepted:
the alternative is a checker that has to decide what a producer is, which is the judgement the rule
hands to a person on purpose.

## 6. The rejected alternatives

**Narrow the certificate arm by source as well as facet.** A `source = 'prober' AND facet =
'certificate'` arm still admits §3's injected row, because `SourceFor` maps `certificate` to
`prober`. A narrowing that keeps the only reachable case is a longer way of writing the defect.

**Keep the arm with a corrected justification.** The justification would have to read *"reserved for a
producer that does not exist"*. An exception that names no writer is the failure mode ADR-1985 §4
builds a deny-list to prevent. Keeping it makes the deny-list an allow-list on one facet.

**Close the recording-side gate instead, and keep both arms.** Dropping a facet-bearing line on a
vantage-less job inside `internal/queue/scopegate.go#authorizedScope.gate` would stop §3's row before
the insert, and it fits ADR-0217's drop-and-log policy better than a constraint violation does. It is
declined here because it leaves the constraint stating a rule wider than the invariant, and a second
mechanism guarding the difference. That is the two-mechanism shape ADR-1985 §3 rejects. The gate
change remains available as a later refinement, and it does not need these arms to stay.

**Edit migration 26200 in place.** It has run in developer databases, so the drop lands as a later
migration that drops and re-adds both constraints. That migration is not this PR: an ADR PR adds one
file.

## 7. What a lying prober costs after the drop

Without the arm, §3's injected row fails the `CHECK`, the completion transaction rolls back, and the
job retries and then dead-letters. ADR-0217 §4 declines to fail a job on a dropped line, because that
hands a compromised prober a denial of service. The cost here is bounded in a way that argument is
not: the prober denies only the job it is already running, which it can deny by returning an error
anyway. It gains nothing over any other job, any other vantage, or any other kind.

The batch's honest lines are lost with it. That is the price of the invariant, and it is paid only on
a job whose prober is already lying about what it measured.

## 8. What reversal costs

Reversal re-adds `OR facet = 'certificate'` to both tables and `OR source = 'zone'` to `span`. No
stored row changes, and no honest writer starts or stops working, because §2 and §4 show that none of
the three arms carries one.

What comes back is a representable NULL-vantage `certificate` row on both tables, reachable from a
prober that lies. The database stops refusing it, and the two reads disagree about it again. That
disagreement is the harm ADR-1985 priced, and
[ADR-0095](./0095-the-aperture-statement-counts-what-the-instrument-cannot-report-not-what-it-did-not-look-at.md)
rules a false aperture claim worse than a missing reading.
