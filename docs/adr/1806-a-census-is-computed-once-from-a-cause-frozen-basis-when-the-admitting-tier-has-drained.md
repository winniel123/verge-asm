---
number: 1806
title: "A census is computed once from a cause-frozen basis, when the admitting tier has drained"
slug: a-census-is-computed-once-from-a-cause-frozen-basis-when-the-admitting-tier-has-drained
date: 2026-09-10
status: accepted
source: grilling
ticket: 1806
map: 1811
proof: {ticket: 1819}
relations:
  - {kind: amends, adr: 26, clause: "2"}
  - {kind: amends, adr: 31}
---

# ADR-1806: A census is computed once from a cause-frozen basis, when the admitting tier has drained

## Decision

A census is computed once, from a basis frozen at its cause, when the tier that admits the subtree has drained.

A fold is one job with one Kind, so a resolution move and the openings beneath it never share a fold, and a census read there is empty. `produceMessages` writes the membership row at the cause and defers the census. The dispatcher's minute poll releases the row once one `hot` dispatch that fanned out after the root's batch has drained. It computes the census, appends the census clause to the headline, and enqueues delivery. A deferred row reaches no panel and no unread badge.

The basis is the root span's own value at that batch, so a later re-point adds no subject to an earlier census.

Rejected: a duration window, which [#27](https://github.com/winniel123/verge-asm/issues/27) refuses in a safety path.

Reversal costs a migration and every message already released.

## 1. Context

[ADR-0031](./0031-membership-alerts-at-the-root-of-the-entering-subtree.md) rules that a membership
message fires once, at the root of the entering sub-tree, and carries the census of what entered
beneath it. Its Decision says that census is *computed once at the cause*.
[ADR-0026](./0026-the-facet-layer-is-evidence-not-a-channel.md) §2 gives the re-point residue the
same shape, over the `Endpoint`s a resolution move opens.

Both read their own fold's `changes`. A fold is one job with one Kind
(`cmd/prober/main.go`, `internal/queue/worker.go`), and the resolution walk emits a `Name` alone
(`internal/measure/resolutionwalk/emit.go`). The `Service` and `Endpoint` spans that a census
counts open in a later `hot` fold. So `membershipCensus` (`internal/queue/produce.go`) and
`rePointResidue` (`internal/queue/repoint.go`) return nothing in production, and ADR-0026 §2's
message never fires. [#1774](https://github.com/winniel123/verge-asm/issues/1774) records the
defect. [#1806](https://github.com/winniel123/verge-asm/issues/1806) grilled the repair.

ADR-0031 recorded the timing as fog: *whether the census is emitted incrementally per completed
`Batch`, or the membership message waits for a defined set of tiers*. This ADR answers it. The
message waits, and the wait is bounded by a drained tier rather than by a clock.

## 2. The mechanism

Root determination stays in the fold. It reads the gap-crossing history that only the fold holds
(`internal/queue/spanfold.go`), so it cannot move to the poll.

1. `produceMessages` writes the membership `message` row at the cause. It carries the entry class,
   the subject, the instant, and a cause-clause headline. It sets a new nullable
   `census_pending_after_batch` that references the root's `batch`. It enqueues no delivery.
2. The dispatcher's existing minute poll (`internal/queue/queue.go`) finds a pending row whose
   first `hot` dispatch after that batch has drained. `dispatch` already carries `scan_id`,
   `created_at` and `status`, so the query needs no new column.
3. The poll computes the census, appends the census clause to the headline, clears the pending
   column, and enqueues delivery.
4. A pending row is not rendered and counts toward no unread badge. Nothing reaches the operator
   until the census is real.

The headline is deferred with the census because it is census-derived. The `message` table states
that the headline is computed once at the cause, and `headline` is `NOT NULL`. A `NULL` `census`
already means *the firing carries only a count, or none*, so it cannot also mean *pending*. The
pending column is therefore a column of its own.

The re-point message gets no pending row. ADR-0026 §2 defines its predicate over the two adjacent
spans alone, and those spans are durable, so the poll re-derives the move from them and fires only
where the residue is non-empty.

## 3. The bound is one drained `hot` dispatch

`ScanHasNonTerminalJobs` (`internal/queue/hotlag.go`) answers whether a dispatch has drained. It is
state the model already holds for the cadence-lag gate.
[#27](https://github.com/winniel123/verge-asm/issues/27) refuses an invented number in a safety
path, and a duration window is one.

The bound names `hot` by Kind. It does not derive the tier from the estate. The reason is recorded
so a later tier change is caught by a reader of this file:

- `hot` is the only connect-outcome tier that ships enabled, and it walks every declared scope
  daily (`internal/scan/hot.go`, `internal/queue/hot.go`).
- `cold` is a disabled superset on the same leaf
  ([ADR-0044](./0044-a-one-off-measurement-has-no-currency.md)), so deriving the tier from the leaf
  would pick a tier that never runs.

The chain is `dns` fold, then `hot` at 24 h, then `http-identity` at 24 h and `tls-acceptance` at
7 d. The last two are gated on the services `hot` found reached, so each needs its predecessor to
drain first.

## 4. The basis is frozen at the cause

"Beneath the root" reads the root span's own value at the root's batch. It never reads live
resolution at release time.

A `Name` root's census is the set of subjects whose address the root's own resolution cites at that
batch, plus the `Endpoint`s that name owns. An `Address` root's census is the set of subjects on
that address. Both axes are already in `subjectBeneathRoot` (`internal/queue/produce.go`).

A frozen basis is what makes the message honest under a second move. Where a name re-points again
before the poll runs, the later address is new ground with a cause of its own, and ADR-0026 §2
answers for it. Reading live resolution would instead fold the second move's subjects into the
first move's message, under the first move's instant.

## 5. What the census still cannot contain

The census carries `reachability` and `certificate`. It carries neither `http-identity` nor
`tls-acceptance`, because those run their own `Scan`s behind the bound of §3.

This is exactly the price ADR-0031 already accepted, in its stated cost that *the entry census
covers only the tier that admitted the subject*, as narrowed by
[ADR-0028](./0028-a-facets-cadence-is-the-cadence-of-its-exchange.md), which put `certificate` on
the reachability exchange. **This ADR keeps that cost and does not move it.** Moving it is a second
decision about which tiers a message waits for, and it would need its own file.

## 6. The two degradations, each with its reason

| Configuration | Behaviour | Why it is honest |
| --- | --- | --- |
| The stale-running reaper is disabled (`staleJobThreshold` zero) | No hold. The census is written at the cause, as before. | `ScanHasNonTerminalJobs` reads a job set that nothing reaps, so one wedged row would hold every message forever. `HotLagGateArmed` (`internal/queue/hotlag.go`) already warns on this configuration, for the same reason. |
| The `hot` `Scan` is disabled | Release at once, with an empty census. | Nothing will ever open beneath the root, so an empty census is accurate rather than premature. |

Neither degradation is silent about itself. Each is a configuration the operator chose, and the
first already carries a warning at dispatch.

## 7. Rejected alternatives

| Alternative | Why not |
| --- | --- |
| A duration window after the cause | An invented number in a safety path, which [#27](https://github.com/winniel123/verge-asm/issues/27) refuses. It is also wrong in both directions: too short drops a slow vantage's openings, and too long delays the product's headline event. |
| Compute the census on every read, derived and never stored | The headline is census-derived and `NOT NULL`, so a derived census makes a sent message's own text mutable after it was sent. A message is one firing of one cause. |
| Hold the delivery on `delivery.run_after` | `run_after` is the retry gate for a delivery that exists. The row would then be a queued send that no operator surface may render, and the panel would need the same pending test anyway. The hold belongs on the message. |
| Reuse a `NULL` `census` as the pending flag | `NULL` already means *the firing carries only a count, or none*. One column would carry two facts, and a released empty census would be indistinguishable from a held one. |
| Wait for `tls-acceptance` as well, so the census is complete | A 7-day hold on the product's headline event, and a second decision on top of this one. §5 keeps ADR-0031's price instead. |

## 8. Proof

[#1819](https://github.com/winniel123/verge-asm/issues/1819) is the proof: a worker-level
integration test across two folds. It folds a `dns` batch that enters a `Name`, folds a `hot` batch
beneath that name's address, drains the dispatch, and asserts that the membership message's census
names the `Service` and the nameless `Endpoint`.

The proof stays a `ticket` proof for the life of this file. That test lands after this PR, and
`docs/spec/adr-governance.md` §3 forbids an ADR amending itself in place, so the field is never
rewritten to a `test` proof.
