# ADR-0226: the flagship read is bounded by the batch's candidate Services, so a per-job message costs what the batch touched

- **Status:** Accepted
- **Date:** 2026-09-07
- **Ticket:** [#1609 flagshipMessages runs two unindexed span scans inside complete's transaction, and they carry the whole 25% drain slowdown](https://github.com/winniel123/verge-asm/issues/1609)
- **Measured by:** [#1588](https://github.com/winniel123/verge-asm/issues/1588) / [PR #1608](https://github.com/winniel123/verge-asm/pull/1608), which named `flagshipMessages` as the site and ruled out `observation` growth
- **Rests on:** [ADR-0064](./0064-a-message-names-what-moved-and-where-nothing-moved-it-says-so.md), which computes a message once, at the cause. The cause is one job's completion, so the read is per job
- **Rests on:** [ADR-0199](./0199-delivery-imports-queue-never-the-reverse-so-the-worker-takes-the-message-enqueuer-as-an-injected-function-that-joins-the-batch-transaction.md), which joins the message write and its `Delivery` rows to the batch transaction. That is why the read stays inside `complete`
- **Rests on:** [ADR-0041](./0041-a-corpus-is-retained-by-what-may-still-read-it-never-by-its-age.md), which never compacts the `span` corpus. A read proportional to that corpus therefore has no ceiling
- **Sibling of, and not ruled by:** [ADR-0105](./0105-inventory-is-a-read-over-the-open-span-corpus-not-a-second-thesis.md). It rules the operator-facing reads over the open-span corpus. Those reads want every Service and keep their unbounded statements

## Context

`flagshipMessages` (`internal/queue/produce.go`) decides whether one Service's internet leg moved
from not-reached to reached. It read every open service reachability span twice. It read once as of
now, through `ListServiceReachabilitySpansByClass`. It read once as of the previous batch's instant,
through `ListServiceReachabilitySpansByClassAt`.

`w.produce` calls it inside `complete`'s transaction, on every completed job.

PR #1608 measured a 1024-job drain and attributed the slowdown. The two statements cost 31.9 ms per
job in the first block of 128 jobs. They cost 597.0 ms per job in the last block. That is 80% of the
705 ms wall-clock rise and 99.7% of the SQL rise. Holding `span` constant flattened the curve
completely and cut the drain from 602 s to 200 s.

**The result set was discarded almost entirely.** `flagshipCandidateServices` derives the Services
the batch touched. `composeInternetLeg` then reads legs for one of those Services at a time. A row
for any other Service was decoded and dropped. One job touches the Services of one address. The read
returned every Service in the estate.

#1609 asked whether a per-job flagship message *may be the real defect*. The measurement below says
the cadence is not the defect. The unbounded read is.

## Decision

> **The two flagship reads take the batch's candidate Service keys and return only those Services'
> spans. The flagship message stays per job, and it stays inside `complete`'s transaction. No index
> ships, because the bound makes the existing indexes sufficient.**

Four limbs.

### 1. The bound is a `subject_key` predicate in the SQL, not a filter in Go

`ListServiceReachabilitySpansByClassForServices` and
`ListServiceReachabilitySpansByClassAtForServices` each take a `service_keys` text array. Each adds
`AND sp.subject_key = ANY(...)` to its predicate. `flagshipMessages` passes the candidate Services
it already derived.

The bound must sit in the SQL. A Go-side filter still transfers and decodes every row, and
§Consequences prices that at roughly 200 ms per job.

The narrowed set is exactly the set the old code used. `composeInternetLeg` is the only consumer of
either result, and it selects by Service key. So the two shapes agree on every message they produce.

### 2. The message stays per job, and it stays inside the transaction

ADR-0064 computes a message once, at the cause. The cause is a job's completion. A batched or
deferred flagship pass would state a move at an instant that is not the instant it happened.

ADR-0199 records the transaction fact. The `Message` row and its `Delivery` rows commit with the
batch, the observations and the spans. A cancelled job rolls all of them back together. A read
hoisted out of that transaction would see spans a rollback later discards, or would miss spans a
concurrent job commits.

So a hoist reopens two accepted decisions and buys nothing the bound does not already buy. The bound
removes the cost. A hoist only moves it.

### 3. No migration and no new index

The bound changes which plan Postgres picks, and the plan it picks needs no new index.

Measured on Postgres 16 with the default 4 MB `work_mem`, over a 616,100-row `span` corpus holding
116,100 open service reachability spans:

| Statement | Old shape | New shape |
| --- | --- | --- |
| `ByClass` | 116,100 rows · parallel `Seq Scan` · external merge sort to disk · 137.6 ms | 15 rows · `Index Scan` on `span_subject_idx` · 0.14 ms |
| `ByClassAt` | 116,100 rows · parallel `Seq Scan` · external merge sort to disk · 122.5 ms | 15 rows · `Index Scan` on `span_subject_idx` · 0.14 ms |

`span_subject_idx` (`db/migrations/19000_span.sql`) leads with `subject_kind`, `subject_key` and
`facet`. The bound supplies an equality on all three, so the index answers both shapes.

A new index would cost a write on every `OpenSpan`, and `OpenSpan` runs in the same hot transaction.
The repair must not pay on the write path for a read the bound already made small.

### 4. The unbounded statements stay, because three web reads need them

`cmd/web/exposure.go`, `cmd/web/deltas.go` and `cmd/web/signals.go` call
`ListServiceReachabilitySpansByClass`. Those screens render every Service's leg, so the corpus is
their answer and not their waste.

Those reads run once per page request, outside any job transaction. They are a different question
about the same rows. This ADR does not rule them, and
[#1625](https://github.com/winniel123/verge-asm/issues/1625) carries them.

## Consequences

A stub-leaf drain measured the repair end to end. PR #1608 §4.2.1 describes that rig. Both runs
drained 1024 addresses against a refusing target. Each recorded the same 134,144 observations and
the same 134,144 spans. Seconds per job, by block of 128:

| Jobs | Before | After |
| --- | --- | --- |
| 1 to 128 | 0.259 | 0.226 |
| 129 to 256 | 0.422 | 0.257 |
| 257 to 384 | 0.618 | 0.203 |
| 385 to 512 | 0.748 | 0.151 |
| 513 to 640 | 0.818 | 0.189 |
| 641 to 768 | 1.018 | 0.173 |
| 769 to 896 | 1.194 | 0.198 |
| 897 to 1024 | 1.361 | 0.210 |
| **Drain** | **825.6 s** | **206.1 s** |

The curve flattens. The last block costs less than the first. This host ran a second Postgres stack
throughout, so the before column is steeper than PR #1608's 602 s baseline. The after column matches
PR #1608's Arm C, which held `span` constant and reached 200 s.

- **The per-job cost stops tracking the corpus.** The bounded read returns the spans of the Services
  one job touched. That count is set by the address, not by how long the install has run.
- **A measurement now names the 138 ms of the rise that sat outside SQL execution time.** The old
  `ByClass` shape cost 227.0 ms end to end against 137.6 ms of SQL execution. The old `ByClassAt`
  shape cost 236.7 ms against 122.5 ms. Roughly 200 ms per job went to wire transfer and Go decode.
  The caller then dropped those rows. `EXPLAIN (ANALYZE)` inflates the execution figure, so the true
  residue exceeds these differences. The bound removes the residue, because it removes the rows.
- **Two generated methods and two row types are new.** `internal/db` gains
  `ListServiceReachabilitySpansByClassForServices` and
  `ListServiceReachabilitySpansByClassAtForServices`, with their own row and parameter types.
- **Four statements now read the same rows through two shapes.** A later edit to the projection must
  change both the bounded statement and its unbounded twin. Nothing enforces that.
- **The three web reads keep the old cost.** Each still scans `span` and sorts to disk on a large
  install. That is a page-render cost outside a transaction.
  [#1625](https://github.com/winniel123/verge-asm/issues/1625) tracks it. No measurement covers
  those renders, so that ticket measures before it changes anything.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** No domain term moves. A flagship message still
  fires on a leg, under [ADR-0029](./0029-an-alert-fires-on-a-leg.md), over the composition
  [ADR-0080](./0080-a-vantage-composition-is-cross-class-or-class-scoped-and-only-one-takes-a-quantifier.md)
  fixes.
- **This repair measured nothing past 1024 jobs.** PR #1608's projection rested on continuing a
  measured slope. This repair extends neither the slope nor the projection.
- **The stub-leaf rig removes the 2.76 s pacer, so it prices the worker and not the estate.** A
  shipped drain pays that leaf cost per address on top of every figure in the table.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **Add an index on `span` and keep the unbounded read** | It changes the constant and not the growth. The read still returns every open service reachability span, so both the row count and the Go decode still rise with a corpus ADR-0041 never compacts. It also pays a write on every `OpenSpan` in the same hot transaction |
| **Hoist `flagshipMessages` out of `complete`'s transaction** | It reopens ADR-0064 and ADR-0199. A message computed after the commit can state a move that a rollback discarded, and ADR-0064 rules that no later pass recomputes it. The whole-corpus read also survives the move, so the cost is relocated and not removed |
| **Compute the flagship message once per drain rather than once per job** | Same objection, in a second form. A batched pass fires at the end of the drain and names an instant that is not the cause's. It also needs its own durable record of what it has already reported |
| **Filter the rows in Go after the unbounded read** | The measurement in §Consequences prices this. Roughly 200 ms per job at 116,100 rows went to wire transfer and decode, before any filter ran |
| **Change the two existing statements in place rather than add bounded twins** | `cmd/web/exposure.go`, `cmd/web/deltas.go` and `cmd/web/signals.go` need every Service. Passing them a key list of the whole estate would rebuild the same scan and add a large parameter to every page render |
| **Bound the read to one Service and call it once per candidate** | A job touching many Services would run many round trips inside the transaction. The array parameter answers all of them in one statement, and the index serves an `= ANY` predicate as well as an equality |
