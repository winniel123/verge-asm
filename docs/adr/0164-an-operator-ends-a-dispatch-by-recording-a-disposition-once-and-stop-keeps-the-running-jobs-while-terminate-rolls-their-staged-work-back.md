# ADR-0164: An operator ends a Dispatch by recording a disposition once, and stop keeps the running jobs while terminate rolls their staged work back

- **Status:** Accepted
- **Date:** 2026-09-05
- **Ticket:** [#1353 ADR gaps: cmd/web/scans.go](https://github.com/winniel123/verge-asm/issues/1353)
- **PR that deleted the comments:** [#1352](https://github.com/winniel123/verge-asm/pull/1352)
- **Not a sub-issue of any map:** [`comment-policy.md`](../spec/comment-policy.md) §8.8
- **Read with:** [ADR-0165](./0165-a-recorded-dispatch-disposition-overrides-the-live-status-derivation-and-the-run-pages-status-word-is-one-token-that-styles-and-labels-the-badge.md), which rules the read side. This ADR rules the write side
- **Bounds:** [`raw-job-output.md`](../spec/raw-job-output.md) §2.4, at §2.4's own site, per [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
- **Rests on:** [ADR-0005](./0005-scan-execution-model.md), which fixes one queue job as one `Batch` and commits the job outcome with its observations. It rules the natural life of a `Dispatch` and never rules ending one

## Context

Six comment blocks in [`cmd/web/scans.go`](../../cmd/web/scans.go) stated this rule. #1352 deleted
them. Their only citation was the token `DF-F4` / `DF-F4b`, which resolves nowhere on disk.
[`comment-policy.md`](../spec/comment-policy.md) §4.7 names that family unrepairable, and §8.3 shape
2 records the gap rather than suppressing it.

**Four on-topic sources were read, and none of them states the rule.**

| Source | What it states | Why it does not suppress |
| --- | --- | --- |
| [`scans-monitor-bounding.md`](../spec/scans-monitor-bounding.md) | The live spec for the `/scans` monitor: a state-chip rollup, a dedicated history window, the split handler read | It never mentions stopping or terminating a `Dispatch`. It predates the acts |
| [`raw-job-output.md`](../spec/raw-job-output.md) §2.4 | *"A mid-flight cancel (`errJobCanceled`) rolls the transcript back with all other staged work — a terminated job discards everything, no exception."* | It rules where a `Transcript` is written. It names no operator act, no disposition token, and not the queue-job cancel that causes the mid-flight cancel |
| [`CONTEXT.md`](../../CONTEXT.md) **Dispatch** | *"One firing of one `Scan` at one scheduled time"*, held for display and operational visibility, fenced out of the comparison path | It names no disposition field |
| [ADR-0005](./0005-scan-execution-model.md) | The execution model: fan-out, overlap, retry, dead-letter, progress grouping | It rules the natural life of a run. It is silent on ending one |

No ADR on disk contains the word *cancel*.

**One further statement exists and is not durable.**
[`db/migrations/22901_scan_cancellation.sql`](../../db/migrations/22901_scan_cancellation.sql)
carries a header comment that states most of this model. SQL is inside the swept corpus
([`comment-policy.md`](../spec/comment-policy.md) §3.6 grades every SQL cell *agent*), so that
comment is sweep-eligible and is not a record. `db/queries/dispatch.sql` carried a second statement
that a later sweep already removed.

### The mechanism, as it stands in the tree

- Both acts are admin-gated beside the trigger: `mux.HandleFunc("POST /scans/stop", s.requireAdmin(s.stopScan))` and the matching `POST /scans/terminate` in [`cmd/web/handlers.go`](../../cmd/web/handlers.go).
- `CancelReadyJobsForDispatch` sets `state = 'cancelled'` where `state = 'ready'`. `CancelActiveJobsForDispatch` does the same where `state IN ('ready', 'running')`. Both are in [`db/queries/dispatch.sql`](../../db/queries/dispatch.sql).
- `SetDispatchStatus` is `UPDATE dispatch SET status = $2 WHERE id = $1 AND status = 'fanned-out'`.
- Migration 22901 admits `'fanned-out'`, `'stopped'` and `'terminated'` on `dispatch.status`, and adds `'cancelled'` to `queue_job.state`.
- `MarkJobDone`, `MarkJobDead` and `MarkJobRetried` in [`db/queries/measurement.sql`](../../db/queries/measurement.sql) each carry `AND state = 'running'` and return their row count.
- In [`internal/queue/worker.go`](../../internal/queue/worker.go), `markDone`, `markDead` and `markRetried` return `errJobCanceled` on a zero row count. `inTx` defers `tx.Rollback` and returns before the commit. `runJobTx` swallows `errJobCanceled` and logs one line.

## Decision

> **An operator ends a `Dispatch` in flight by recording a disposition on the `Dispatch` row.
> `stopped` cancels the pending jobs alone and lets the running jobs finish and commit.
> `terminated` cancels the running jobs as well, and each cancelled worker's guarded terminal write
> then rolls its staged work back. A disposition is written once over `fanned-out`, and never over
> another disposition. Neither act deletes committed work.**

Five limbs.

### 1. `fanned-out` is the absence of an operator act

`dispatch.status` holds three tokens. Two of them are operator-minted. `fanned-out` is what the fan
-out writes and what a natural run keeps to the end. It is not a third disposition and it is not a
terminal outcome. A reader who wants to know whether an operator ended a run asks whether the status
is still `fanned-out`.

### 2. Stop is graceful, and the ground is the claim predicate

`stopScan` cancels the `ready` jobs alone. The ground is that `ClaimJob` selects `state = 'ready'`
by itself, so a cancelled pending job leaves the claimable set the instant it is marked. No worker
has to be told and no worker has to check.

A running job is untouched. It finishes its probe, commits its batch and its observations, and
reaches `done` or `dead` on the ordinary path. Nothing already observed is discarded. The operator
gets *stop enqueuing new work*, not *abandon work in progress*.

The toast reports both halves, because both are real: the count of pending jobs cancelled, and the
count of running jobs still finishing.

### 3. Terminate discards staged work, and staged work only

`terminateScan` cancels the `ready` **and** the `running` jobs. A cancelled running job is the one
case where the worker loses work.

The mechanism is a guarded terminal write. `MarkJobDone`, `MarkJobDead` and `MarkJobRetried` all
guard on `state = 'running'`. A cancelled job is `cancelled`, so the write affects no row. The
worker reads the zero row count, returns `errJobCanceled`, and `inTx` rolls the transaction back.
The staged batch, its observations and its `Transcript` go with it.
[`raw-job-output.md`](../spec/raw-job-output.md) §2.4 rules the `Transcript`'s share of that
rollback.

**The bound on the loss is "staged".** Both cancel queries update `queue_job.state` and nothing
else. Neither one deletes a `batch` row and neither one deletes an `observation` row. A running job
whose transaction already committed is `done` or `dead` before the cancel reaches it, and its work
stands. So a terminate loses the attempt in flight and never rewrites the estate record behind it.

`runJobTx` then swallows `errJobCanceled` rather than returning it. The cancellation already
recorded the job's terminal state, so the worker owes nothing more.

### 4. The disposition is written once, and the first one stands

`SetDispatchStatus` guards on `status = 'fanned-out'`. A second act against the same `Dispatch`
records nothing, and the first disposition survives.

**This ADR states the cost rather than smoothing it.** An operator who stops a `Dispatch` and then
terminates it gets the terminate's job cancellations, because `CancelActiveJobsForDispatch` carries
no such guard. The row still reads `stopped`. The record then understates what happened.

The once-only write is kept, for two reasons. It makes the recorded disposition a statement about
the operator's first decision rather than the last one, and it forbids a later write from moving a
`Dispatch` back to `fanned-out`. Escalation from stop to terminate is **not expressible today**. A
later ticket that wants it must widen the guard deliberately and rule the transition, rather than
drop the guard.

> **The cost is filed as a defect, [#1421](https://github.com/winniel123/verge-asm/issues/1421).**
> This limb rules the guard, and the guard is right. What #1421 owns is that the **cancel** is not
> guarded to match it, so one act's two halves disagree: the jobs are cancelled, the disposition
> records nothing, and the toast still says *"Scan terminated"*. The escalation is reachable from
> the console, because `ListActiveDispatchProgress` filters on job state alone and never reads
> `d.status`, so a stopped `Dispatch` with draining jobs keeps its **Terminate** control. Whichever
> fix lands, this limb is amended with it.

### 5. Both acts are admin-gated, exactly as the trigger is

Ending a run in flight is the same class of act as firing one. A non-admin `POST` is refused before
either handler runs.

## Consequences

- **[`raw-job-output.md`](../spec/raw-job-output.md) §2.4 gains one bounding sentence** at its own
  site, naming this ADR. Its *"discards everything, no exception"* is true of the attempt's staged
  work. A reader who arrives holding §2.4 and not this ADR could read it as reaching committed
  observations. [ADR-0058](./0058-a-superseded-mechanism-is-withdrawn-at-the-site-that-specifies-it.md)
  requires the edit at §2.4 and not only here.
- **[`cmd/web/scans.go`](../../cmd/web/scans.go) carries three citations.** `stopScan`'s claim
  -predicate comment and `terminateScan`'s staged-work comment each gain this ADR's number.
  `terminateScan` gains one new comment at its `SetDispatchStatus` call, because the once-only guard
  lives in SQL and no reader of the Go call site can recover it.
- **`dispatchOutcome` gains nothing.** Its rule is its own switch body, so a comment there restates
  the code.
- **No production behaviour changes.** The code already has the shape this ADR states. The change is
  that the shape now has a record.
- **[`CONTEXT.md`](../../CONTEXT.md) gains nothing.** The **Dispatch** entry already says the corpus
  exists *"for display and operational visibility"*, and a disposition is exactly that. It states no
  clause this narrows, and it asserts nothing about the estate that a disposition changes. The
  comparison-path fence is untouched, because the drift engine reads neither `dispatch` nor
  `queue_job`.
- **A third disposition needs this ADR and a migration.** `dispatch_status_check` admits three
  tokens. A fourth act must widen the constraint, and it must say how limb 4's once-only write
  treats it.
- **A `Dispatch` ended by an operator can never reach 100% progress.** A cancelled job counts in the
  denominator and never in the numerator. [ADR-0165](./0165-a-recorded-dispatch-disposition-overrides-the-live-status-derivation-and-the-run-pages-status-word-is-one-token-that-styles-and-labels-the-badge.md)
  rules what the page shows instead.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| **One act only — a single "cancel" that kills everything** | It prices the two operator intents the same. An operator who wants to stop a runaway fan-out usually wants the vantages that already answered to be kept. Collapsing the acts makes every stop cost the in-flight attempts, and it makes the graceful case unreachable |
| **Stop by deleting the pending job rows instead of marking them `cancelled`** | It loses the count the toast reports and the evidence that the jobs existed. The rollup and the run page both read `queue_job`, so a deleted row silently shrinks the denominator and makes a stopped run look complete |
| **Terminate by killing the worker process** | It is not addressable from a web handler, it ends every unrelated job the worker holds, and it produces no record. The guarded terminal write reaches exactly the jobs of one `Dispatch` and needs no channel to the worker at all |
| **Let a terminate delete the batches its jobs already committed** | It rewrites the estate record to match an operator's later change of mind. A committed batch is evidence that a measurement ran, and ADR-0005 commits it with the job outcome for that reason. Limb 3 bounds the loss to the attempt in flight |
| **Have the worker poll for a cancel flag** | It adds a read to every job's hot path and a window in which a cancelled job still commits. The guarded write already gives the exact property, at the moment the write happens, with no polling |
| **Allow a disposition to overwrite an earlier one** | It lets a later write move a `Dispatch` back to `fanned-out` and erase the fact that an operator ended it. Limb 4 keeps the guard and states the escalation cost instead |
| **Record the disposition per job rather than on the `Dispatch`** | The jobs already carry it: a cancelled job is `cancelled`. The `Dispatch` needs the fact that a **person** ended the run, which no job state can express, because a job is also cancelled by the other act |
| **Leave the rule in `db/migrations/22901_scan_cancellation.sql`'s header** | SQL is in the swept corpus and that comment is sweep-eligible. A migration is also read once, at the moment it runs. It is not where a later session looks for a rule about a screen |
| **Fold the rule into [ADR-0005](./0005-scan-execution-model.md) as an amendment** | ADR-0005 is silent here rather than wrong here. Under ADR-0058 an amendment carries a claim about the world, and this is a new act on a corpus ADR-0005 ruled the natural life of. ADR-0005 takes no edit |
| **File one ADR covering the write side and the read side together** | Two rules with different scopes. The write side binds the worker, the queue schema and the migration. The read side binds the run page and its template. A later session applying one should not have to read the other |
