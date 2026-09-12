---
number: 1851
title: "A dispatch records that its fan-out finished, and a later tick retires one that never did"
slug: a-dispatch-records-that-its-fan-out-finished-and-a-later-tick-retires-one-that-never-did
date: 2026-09-12
source: fix
status: accepted
ticket: 1851
map: 1811
proof: {test: "internal/db/message_release_test.go::TestTheFanOutMarkIsWrittenOnceAndCarriesNoInstant"}
relations:
  - {kind: amends, adr: 1806, clause: "6"}
  - {kind: rests-on, adr: 164, clause: "1"}
---

# ADR-1851: A dispatch records that its fan-out finished, and a later tick retires one that never did

## Decision

A `dispatch` carries a boolean recording that its fan-out finished. Both release predicates read
that column for the first half of ADR-1806 §3's bound, in place of the job-existence proxy #1816
used.

A streamed tier commits its dispatch row before its jobs, so a job count cannot separate a fan-out
still streaming from one that finished with nothing to enqueue. A `hot` tier that is enabled and
admits nothing is the second case, and the proxy held its membership message forever.

A crashed fan-out marks itself never. The next claimed `hot` tick marks it abandoned in a second
column, and the bound skips such a row. A tick is a cadence, so no clock enters the predicate.

Rejected: an age bound on a jobless dispatch, which ADR-1806 §7 and
[#27](https://github.com/winniel123/verge-asm/issues/27) both refuse.

Reversal costs a migration and every census already computed.

## 1. Context

ADR-1806 §3 states the release bound in two halves: *the dispatch fanned out and every `queue_job`
it enqueued has reached a terminal state*. `CONTEXT.md`'s `Drained` entry states the same, and names
what went wrong: *both halves carry weight, and the model held only the second*.

Only the second half had a representation. `TryFanOut` inserts the dispatch row with
`status = 'fanned-out'` before any job exists, and `hot`, `cold` and `edge-fanout` stream their jobs
in later chunks through `streamEnqueue`. So the status column is written at claim time and cannot
answer whether the fan-out finished.

[#1816](https://github.com/winniel123/verge-asm/issues/1816) closed the resulting window with a
proxy: a dispatch with no job at all is mid-fan-out, so hold. That is correct while every fan-out
enqueues something.

[#1851](https://github.com/winniel123/verge-asm/issues/1851) records where it is not. A `hot` tier
that is enabled and admits nothing — every candidate address refused by `Custody`, or no `Vantage`
provisioned — finishes its fan-out with an empty job set. The proxy reads that as *not yet* forever,
and because the bound is the *first* such dispatch, no later one substitutes. The held membership
message reaches no panel and no unread badge, so the hold is silent.

[#1817](https://github.com/winniel123/verge-asm/issues/1817) tried the column-free repair — skip a
jobless dispatch when choosing the first — and reverted it, because a later dispatch could then
answer for an earlier one still mid-fan-out.

## 2. The mark is a column on dispatch

`dispatch` gains `fanout_complete`, a boolean. A streamed tier writes it **inside** the transaction
that commits its last chunk, and a fan-out with no job at all opens one transaction to carry the
mark alone. An atomic tier writes it inside the transaction that commits its jobs, where the row and
its jobs were always consistent.

Writing it after that transaction instead would leave a gap. The dispatch is then finished,
unmarked, and — once its jobs reach a terminal state — indistinguishable from one abandoned mid-fan
-out, so §3's arm would retire a healthy run. One transaction closes the gap, and it also means a
failed mark rolls its own chunk back rather than leaving a finished fan-out unmarked for a cadence.

Both predicates then test the column instead of counting jobs. The bound still names the *first*
`fanned-out` `hot` dispatch at or after the root's batch, so an unfinished fan-out holds the row and
no later dispatch answers for it. That is #1816's window, closed by the same shape and for the same
reason.

The column holds no instant. A timestamp would answer the same question, and it would put an
instant one step from a duration test in a release predicate — the test ADR-1806 §7 and #27 both
refuse. A boolean cannot be compared to a clock.

Existing rows backfill `true`. Their fan-out is not in flight, and a `false` would hold every
already-pending message behind a dispatch that nothing will ever mark.

## 3. A tick retires an abandoned fan-out

A streamed fan-out that crashes between chunks marks itself never, so its dispatch would hold the
bound for good. Before this ADR such a crash left a `fanned-out` row with partial jobs, which
drained and released the message with an under-covered census. The column converts that silent
under-coverage into a permanent hold, so the arm is owed.

When a later `hot` tick is claimed, an earlier `fanned-out` `hot` dispatch with the mark unset gains
a second column, `fanout_abandoned`. The inner select that picks the first `hot` dispatch skips an
abandoned row, so the bound moves to the next live one. A tick is a cadence rather than a duration,
so the predicate stays clock-free.

**The mark is a column and not a fifth `dispatch.status` token.** ADR-0164 §1 rules that `status`
carries the operator's disposition, and its
[#1523](https://github.com/winniel123/verge-asm/issues/1523) amendment prices a machine-minted
token: `stopped` and `terminated` each cancel jobs and mean a person acted, `dispatchOutcome` renders
every token it names, and that word is also a CSS class (ADR-0165 §1). Abandonment cancels nothing,
names no person, and is read by no operator surface. Reusing `terminated` would report a machine
retirement as an operator act and would refuse the operator's own Stop, whose guard admits
`fanned-out` alone. So the fact takes a column, and §6's boolean argument holds for both halves.

**This is not [#1817](https://github.com/winniel123/verge-asm/issues/1817)'s reverted shape.** That
one skipped a dispatch for holding no job, which a dispatch mid-fan-out also does, so a later
dispatch could answer for an earlier one still streaming. This one skips a dispatch for a **recorded**
fact. Nothing infers abandonment, so no streaming dispatch is passed over and #1816's window stays
closed.

**The arm is `hot` alone.** `cold` and `edge-fanout` stream too, and `hotLagGateApplies` covers
neither, so two of their fan-outs may overlap by design and neither may retire the other. The
release bound names `hot` by Kind (ADR-1806 §3), so `hot` is also the only tier that needs the arm.

Three clauses keep a live `hot` fan-out safe. The tick excludes the dispatch it just claimed, by id.
An already-abandoned row is not re-marked. And a fan-out still streaming holds `ready` jobs, which
the marking query refuses to cross — the same state the cadence-lag gate of ADR-0137 §4 reads one
statement earlier to skip the tick outright, which is why the arm is confined to the tier that gate
covers.

One window remains and is recorded rather than closed: a fan-out claimed but yet to flush its first
chunk holds no job, so a tick racing it could mark a live dispatch abandoned. The jobs it then
enqueues still carry their `batch_id` and still run, so no measurement is lost, and the bound
answers from the newer dispatch. Closing the window wants a fan-out lock held across chunks, which
prices a pinned connection against a race whose width is one chunk.

## 4. What ADR-1806 §6 row 2 overstated

§6 row 2 reads *release at once, with an empty census* where the `hot` `Scan` is disabled. The
release is right and the census clause overstates it.

The census read is bounded below by the root's own batch, inclusive
([#1816](https://github.com/winniel123/verge-asm/issues/1816)), so a released row still names
whatever the root's own fold opened beneath it. For a `Name` root that is usually nothing, because
its `Service` and `Endpoint` open in a later `hot` fold. So the census §6 describes is reached by
**reading** and never by assertion, and no arm of either predicate forces it.

This ADR adds no arm. The two degradations of §6 stand as they are, and the repair of §2 sits beside
them in the drain test rather than in a third disjunct.

## 5. Proof, and what it does not cover

The proof is a SQL-text assertion. `internal/db` has no Postgres-backed test, which is why
ADR-1806 took a `ticket` proof for its own mechanism.

`TestTheFanOutMarkIsWrittenOnceAndCarriesNoInstant` holds the mark to a boolean and forbids an
instant. Three assertions beside it carry the rest. The predicate tests in `message_release_test.go`
and `repoint_settle_test.go` require `first_hot.fanout_complete`.
`TestAnAbandonedDispatchIsRetiredByATickAndNeverByAClock` holds §3's arm off both a clock and an
operator disposition. `TestTheBoundPassesOverAnAbandonedDispatch` requires the inner select to skip
an abandoned row, without which the mark moves nothing.

Two limits are worth stating rather than leaving to a reader. The guard that forbids the old
job-count proxy catches one spelling of it and not a rewrite, so the positive assertion is the
load-bearing half. And no offline test covers the behaviour: that a jobless-but-finished `hot`
dispatch releases a held row, and that an abandoned dispatch moves the bound. That wants the
two-fold worker integration test
[#1819](https://github.com/winniel123/verge-asm/issues/1819) already owes ADR-1806.

## 6. Rejected alternatives

| Alternative | Why not |
| --- | --- |
| An age bound on a jobless dispatch | An invented number in a safety path, refused by #27 and by ADR-1806 §7. It is also wrong in both directions: too short releases before a slow vantage's openings land, too long delays the product's headline event. |
| A fifth `dispatch` status, so `status = 'fanned-out'` means literally what ADR-1806 §2 says | The honest-looking shape, and it costs more than it returns. ADR-0164 §1 rules `status` as the operator's disposition, so a fifth token amends that count and owes the #1523 amendment's three grounds and its transition table. `dispatchOutcome` and `runStatusLabel` gain a value, and that word is also a CSS class (ADR-0165 §1), so a design-system change rides a fact no operator reads. Two booleans move no operator surface. |
| Write `terminated` on an abandoned fan-out, reusing a token that already exists | Refused after review. ADR-0164 §1 makes `terminated` mean an operator acted **and** the running jobs were cancelled **and** their staged work rolled back. This arm does none of the three. It would also report a machine retirement as an operator act on the run page, and `SetDispatchStatus`'s guard admits `fanned-out` alone, so the operator's own Stop would then match no row while still reporting success. |
| A completion timestamp instead of a boolean | It answers the same question and stores an instant the predicate does not read, which invites the duration test §2 exists to prevent. |
| Skip a jobless dispatch when choosing the first | #1817 tried it and reverted it. A later dispatch answers for an earlier one still mid-fan-out, which re-opens #1816's window. |
| Let the stale-job reaper retire an abandoned fan-out on `staleJobThreshold` | A duration would then decide a release, one hop from the predicate §7 keeps clock-free. It also couples the arm to a dial ADR-1806 §6 row 1 already reads for a different purpose. |
| Rule the degradation acceptable and amend §6 row 2 to say so | #1851's first acceptance line refuses it: an enabled tier that enqueues nothing must not hold a row forever. A surface that says *we are holding this and will never stop* is not a repair. |
