---
number: 1867
title: "With no reaper the re-point residue names the ground the at-cause census could not"
slug: with-no-reaper-the-re-point-residue-names-the-ground-the-at-cause-census-could-not
date: 2026-09-13
source: fix
status: accepted
ticket: 1867
proof: {test: "internal/queue/repointsettle_test.go::TestWithNoReaperTheResidueNamesTheGroundTheRootCouldNot"}
relations:
  - {kind: amends, adr: 1806, clause: "6"}
  - {kind: rests-on, adr: 26}
---

# ADR-1867: With no reaper the re-point residue names the ground the at-cause census could not

## Decision

**When the stale-running reaper is disabled, the re-point residue waits on the hot fan-out and names
every Endpoint beneath an address new to the estate, except the ones its own fold's roots named.**

ADR-1806 §6 row 1 writes a root's census at the cause. A fold is one job with one Kind, so the `dns`
fold that moved the Name opens no Endpoint, and that census is empty. ADR-0026's residue then drops
the same ground, and its settle short-circuited to the next poll pass. Together they announced the
subtree to nobody. The residue now suppresses only what a root counted, which `in_fold_batch`
reports per subject, and its fold settles once the first `hot` dispatch after it fans out.

Rejected: leaving the loss and widening the dispatch warning. An operator cannot act on a subtree no
message names.

Reversal costs the query column, the settle-path thread, and every message already fired.

## 1. Context

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) fires one membership
message at the root of an entering sub-tree, carrying the census of what entered beneath it.
[ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md) gives a resolution move a residue of
the same shape, over the `Endpoint`s the move opened.

The two partition one ground. A re-point onto an address new to the estate roots on that Address, and
the residue drops what the root already names. `rePointResidue` (`internal/queue/repoint.go`) reads a
freshness test for that drop, and `addressesNewToEstate` is the same test the root reads.

[ADR-1806](./1806-a-census-is-computed-once-from-a-cause-frozen-basis-when-the-admitting-tier-has-drained.md)
then held the root's census until the admitting tier drained, because a `dns` fold opens nothing
beneath the address it enters. A held census counts the whole subtree at release, so the partition
still holds.

## 2. The configuration that breaks the partition

ADR-1806 §6 row 1 refuses the hold when the stale-running reaper is disabled. Its reason stands: the
drain test reads a job set nothing reaps, so one wedged row would hold every message forever. The
root falls back to a census written at the cause.

§6 weighed that fallback against the root alone, and called an empty census the price. It did not
reach the residue. So on this one configuration both halves of the partition go quiet:

| Half | With a hold | At the cause |
| --- | --- | --- |
| Address root | Counts the subtree at release | Counts its own fold, which opened nothing |
| Re-point residue | Drops the ground the root covers | Drops the same ground, uncovered |

The dispatch warning (`internal/queue/hotlag.go`) already told the operator that each half is
narrowed. It did not say that the two together name no endpoint at all.

## 3. What a root still covers

The residue cannot drop the whole address, and it cannot name the whole address either. A census
written at the cause counts what that fold opened beneath the root, so the residue owes the operator
the rest and nothing more.

`ListSubjectsOpenedSinceBatch` (`db/queries/span.sql`) already reads from the fold's own batch
inclusive. It now reports, per subject, whether that subject opened in the fold's own batch. The
residue suppresses a subject when the root above it held its census, or when the fold's own batch
opened it.

One rule serves both root kinds, because one `holdCensus` serves both root producers
(`internal/queue/produce.go`). The Address root reads the freshness test, and a Name root opened in
the same fold reads `coveredByFoldRoot`. On this configuration both wrote at the cause, so both
cover their own fold alone.

This states the rule rather than resting on the emptiness of a `dns` fold. A fold that both moves a
Name and opens an `Endpoint` beneath the new address would double-name it under the weaker rule.

The hold is read from the row, not from the knob the fold read. §4 makes the settle wait a `hot`
cadence, so a restart can change `VERGE_STALE_JOB_TIMEOUT` in between. `BatchHeldItsRootCensus`
(`db/queries/span.sql`) answers what the fold actually did, so a knob moved after the fact neither
double-names the subtree nor silently restores the gap.

## 4. When the fold settles

A rule about what the residue names is worth nothing if the residue is read before the ground
exists. `ListSettleableRePointBatches` (`db/queries/measurement.sql`) took the reaper's state as a
whole disjunct, so on this configuration every fold settled on the dispatcher's next minute tick.
The `hot` tier opens the subtree on its own cadence, which ships at 86400s. The residue read an
empty subject set, `message.RePoint` refused it, and the fold was never read again.

The settle now drops the drain test alone, and keeps the fan-out-finished test on either
configuration. §6 row 1's wedge argument is about the drain test: it reads a job set nothing reaps.
The fan-out fact is recorded, and ADR-1851 §3 records abandonment, so an unfinished fan-out is
skipped rather than waited on. Nothing can wedge the fold.

This differs from the held-message release path, which still leaves at once. A held row that never
releases holds every message forever. A batch flag holds nothing: a fold that settles early fires
no message, which is the loss this ADR closes.

## 5. Rejected alternatives

**Leave the loss, and widen the dispatch warning.** Cheapest, and it keeps one reading in the settle
path. It was refused because a warning at dispatch does not reach the operator who reads the message
panel a day later, and the subtree that entered has no other carrier.

**Arm the hold on the Address root whatever the reaper's state, with a bound of its own.** This
contradicts §6 row 1's stated reason. With nothing reaping the job set, the new bound would be the
only thing that ever releases a held row, which is the failure §6 row 1 exists to refuse. §4 waits
on a batch flag instead, which owes no message and so can lose none.

**Rest on the invariant that a `dns` fold opens no `Endpoint`.** The resolution walk emits a `Name`
alone (`internal/measure/resolutionwalk/emit.go`), so dropping the freshness test outright is correct
today. It was refused because nothing in the settle path states that invariant, and a later Kind that
opened both would double-name the subtree with no test to catch it.
