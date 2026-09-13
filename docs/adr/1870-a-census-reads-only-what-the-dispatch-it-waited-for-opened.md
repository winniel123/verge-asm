---
number: 1870
title: "A census reads only what the dispatch it waited for opened"
slug: a-census-reads-only-what-the-dispatch-it-waited-for-opened
date: 2026-09-13
source: fix
status: accepted
ticket: 1870
proof: {test: "internal/queue/censusproof_test.go::TestADelayedReleaseNamesNoSubjectOpenedAfterTheDrainedDispatch"}
relations:
  - {kind: amends, adr: 1806, clause: "3"}
  - {kind: rests-on, adr: 1809, clause: "4"}
---

# ADR-1870: A census reads only what the dispatch it waited for opened

## Decision

**The release read carries an upper bound. It is the last batch of the `hot` dispatch whose drain
released the row. Where no dispatch drained, it is the root's own batch.**

The read had a lower bound only. A release delayed past later folds therefore named every subject
those folds opened, in a message stamped with the first fold's instant. ADR-1806 §4 freezes the
basis against that outcome on the citation axis. The open top reached it on the subject axis.

The bound is a batch id and never an instant, because ADR-1806 §8 records that no column orders a
dispatch against a batch soundly.

Rejected: capping the census length, which ADR-0031 and ADR-0102 both bar. Rejected: measuring
first, which answers cost and not authorship.

Reversal costs every census computed under the bound.

## 1. Context

[ADR-1806](./1806-a-census-is-computed-once-from-a-cause-frozen-basis-when-the-admitting-tier-has-drained.md)
§3 holds a membership message until one `hot` dispatch has drained, then computes its census.
`releaseHeldMessage` (`internal/queue/release.go`) reads the subjects through
`ListSubjectsOpenedSinceBatch` and filters them with `censusBeneathRoot`.

That query bounded the read below and not above:

```sql
WHERE s.opened_batch_id >= sqlc.arg(batch_id)::bigint
  AND s.subject_kind IN ('service', 'endpoint');
```

[#1816](https://github.com/winniel123/verge-asm/issues/1816) records why the lower bound is
inclusive. Nothing states a top.
[#1774](https://github.com/winniel123/verge-asm/issues/1774)'s closing comment named the read as
unbounded and left it.
[ADR-1809](./1809-a-name-roots-census-counts-on-the-address-it-cites-not-beneath-the-name.md) §4
named it again, as volume, and declined it.

[#1870](https://github.com/winniel123/verge-asm/issues/1870) asked what a census may cost. The
answer here is not about cost.

## 2. An open top folds a later cause into an earlier message

ADR-1806 §4 states the hazard in its own words. Reading live resolution "would instead fold the
second move's subjects into the first move's message, under the first move's instant". The frozen
basis stops that, and it stops it on one axis: which addresses the root cites.

Membership has a second axis. `subjectBeneathRoot` admits a subject whose address the frozen basis
cites, whenever that subject appears in the read. The read was every subject opened since the root's
batch, with no end. So a `Service` or an `Endpoint` opened by a dispatch months later, on an address
the basis still cites, entered the census of a message whose instant is the first fold's.

The proof holds the exact case. A `dns` fold enters `example.net`, a `hot` fold opens the `Service`
and the nameless `Endpoint`, and a later `http-identity` and `tls-acceptance` fold enters the named
`Endpoint`. Released after that later fold, the census named three subjects. Two of them are the
cause. The third is a fold the message never waited for and never announced.

This is authorship, not volume. A shorter census is a consequence and not the reason.

## 3. The bound is the dispatch that answered the release

`ListReleasableHeldMessages` already chooses one dispatch: the first `hot` dispatch at or after the
root's batch that completed its fan-out and drained. That dispatch is what the hold was waiting for,
so it is what the census is about. The query now yields the last batch of that dispatch as
`census_upper_batch`, from a `LEFT JOIN LATERAL` over the same `first_hot` selection the release arm
reads. One subquery fixes both, so the bound and the release cannot drift apart.

It reads `batch.dispatch_id`, which is set membership and exact, and
`db/migrations/26100_batch_dispatch_index.sql` indexes that column so the read is not a sequential
scan of `batch` once per held row. It does not compare `created_at` columns. ADR-1806 §8's second open question records that such a comparison is unsound here, because
`InsertBatch` stamps the batch at the top of the fold transaction while the spans commit at its end.
A batch id needs no instant.

Two cases fall back to the root's own batch, and `COALESCE` writes both:

- **The drained dispatch opened no batch.** ADR-1851 §2 records this configuration: a `hot` tier
  that is enabled and admits nothing finishes its fan-out with an empty job set. Nothing opened
  beneath the root, so the root's own fold is the whole census.
- **No dispatch drained.** ADR-1806 §6's two degradations release without one. Its table already
  calls for the census "at the cause", and the root's own batch is that census.

A retired dispatch needs no rule. Dispatch retention deletes the row, so `first_hot` selects the
next dispatch rather than a row whose batches were orphaned to `NULL`. The bound then widens, and
never narrows.

## 4. What this does not bound

Two reads keep the shape they had.

**The re-point residue.** `settleRePointFold` reads the same query for ADR-0026 §2's message. That
message is not this one, and ADR-1809 §4 records that bounding it is a decision of its own. The
upper bound is optional in the query for this reason, and the residue passes none.

**The `k` multiplier.** #1870's second remedy computes one census per address rather than per root,
which would remove it. That needs a release rule considering every pending root on an address
together, and the claim in ADR-1806 §2 takes one row at a time.
[#1774](https://github.com/winniel123/verge-asm/issues/1774)'s triage found the ordering hard. `k`
messages still carry their own census here. Only `n` is bounded.

## 5. Rejected alternatives

**Truncate the census.** [ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md)
puts every count in the payload, and
[ADR-0102](./0102-a-subjects-row-is-the-base-a-census-member-row-is-its-explicit-modifier.md) locks
a member list's header count to `list.length`, exact. A truncated list reports a number no reader
can reach. [#27](https://github.com/winniel123/verge-asm/issues/27) refuses an invented number in a
safety path.

**Measure first.** #1870's third remedy records that no install has shown the real `k` or `n`, and
proposes instrumentation before a change. Measurement answers what the read costs. It does not
answer whether a later fold's subject belongs in an earlier message, and ADR-1806 §4 already
answered that.

**Bound on an instant.** A `created_at` window would need the ordering ADR-1806 §8 records as
unanswered, and #27 refuses a duration window in a safety path.

## 6. Proof

`TestADelayedReleaseNamesNoSubjectOpenedAfterTheDrainedDispatch`
(`internal/queue/censusproof_test.go`) folds the three batches, releases after the last, and asserts
the census names the drained dispatch's two subjects and not the named `Endpoint` the later fold
entered. Removing the bound fails it on that subject.

`TestTheCensusReadStopsAtTheDrainedDispatch` and `TestTheReleaseRowCarriesTheUpperBound`
(`internal/db/census_read_bound_test.go`) hold the query shapes.
`TestTheResidueReadsNoUpperBound` (`internal/queue/repointsettle_test.go`) holds §4's first half.
