# ADR-0118: report scheduling dispatches on a computable window, and the receipt is its own dispatch record

- **Status:** Accepted
- **Date:** 2026-08-24
- **Ticket:** [#502 On-cadence report dispatcher](https://github.com/winniel123/verge-asm/issues/502)
- **Map:** [#499 report dispatch + delivery](https://github.com/winniel123/verge-asm/issues/499)

## Context

A `report_schedule` (migration 21700) is a declared, recurring report: a name, a set of
sections, a **cadence label**, and a format. Until now it was inert data — the "Run now"
row action ([reports_schedule.go](../../cmd/web/reports_schedule.go)) was the only thing that
turned a schedule into a run, cutting the artifact for the current period and stamping a
`report_delivery` receipt (migration 22500, [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)).
Nothing fired a schedule **on its cadence**. T3/#502 adds the loop that does.

Two questions had to be settled to build it honestly.

**1. What is a cadence, operationally?** The cadence is stored as a free-text label —
a lower-cased preset (`every 6h`, `daily · 08:00`, `weekly · mon 09:00`, `monthly · 1st`)
or, for the wizard's "Custom…" option, an operator-authored cron string. The ticket flagged
[ADR-0091](./0091-the-routing-unit-is-the-class-and-the-cause-is-refused-as-a-routing-key.md)'s
concern: an operator-authored expression evaluated as a predicate over a versioned rule set is
exactly what ADR-0091 forbids. Does a cron-driven scheduler cross that line?

**2. Where does dispatch idempotency live?** The queue dispatcher
([internal/queue](../../internal/queue/queue.go)) fans a Scan out once per cadence window,
keyed on a floored `(scan, scheduled_time)` tick under a per-scan advisory lock: a second poll
inside one window conflicts and is a recorded skip, never a second fan-out. A report dispatcher
needs the same guarantee — a minute-poll loop must not cut the same report twice in one window.
The open question was whether that needs a new `report_dispatch` table beside `report_delivery`.

## Decision

> **A schedule dispatches on a coarse, model-owned *window* computed from its cadence — never
> by evaluating the operator's cron expression — and the `report_delivery` receipt is itself the
> dispatch record, made idempotent by a nullable `scheduled_tick` and a partial-unique key,
> not by a separate table.**

Three parts.

### 1. Cadence normalises to a closed window vocabulary — ADR-0091-clean

`report.CadenceWindow` ([internal/report/cadence.go](../../internal/report/cadence.go)) is the
single source of truth that maps a cadence label to one of four coarse windows: **6h / daily /
weekly / monthly**, defaulting to **weekly** for anything unrecognised. It is moved verbatim
from the Run-now handler's `reportCadenceWindow`, so a manual run and a scheduled run of the
same schedule compute the identical window from one function.

The dispatcher dispatches on that window and **never interprets a cron predicate**. A custom /
unrecognised cadence falls to the weekly window — exactly as Run-now already treated it. This is
why ADR-0091 does not bite: the window vocabulary is a **closed, model-owned set**, and no
operator-authored expression is ever evaluated as a predicate over a versioned rule set. A cron
string is not run; it is bucketed. ADR-0091's specific hazard — a mutable operator expression
silently deciding a model-relevant outcome — is absent, because the outcome (which of four
windows) is decided by model-owned string matching, not by the operator's text. The cost is
honestly stated: a custom cron is approximated as weekly rather than honoured to the minute.
Minute-accurate custom cadences are a later refinement (a real cron evaluator would reopen the
ADR-0091 question deliberately), not a silent predicate today.

### 2. The receipt IS the dispatch record — a nullable tick + partial-unique key

There is **no separate `report_dispatch` table**. A scheduled run's `report_delivery` receipt
already IS the one-run record, so migration 22600 adds one nullable column to it:

- `scheduled_tick TIMESTAMPTZ` — the floored cadence boundary the run was dispatched for
  (`internal/report` `scheduledTick`, copied from queue so the report side floors identically
  to the measurement side). **NULL** on a manual "Run now" receipt: a manual run is keyed to the
  instant the operator asked and never contends on a tick.
- A **partial unique index** `(schedule_id, scheduled_tick) WHERE scheduled_tick IS NOT NULL`.
  This is the on-cadence idempotency backstop, mirroring the queue dispatcher's unique
  `(scan, scheduled_time)`: the first poll in a window inserts and wins the claim; a later poll
  conflicts and returns no row (a recorded skip, not a double-run). The partial predicate keeps
  every NULL-tick manual receipt out of the index, so manual runs stay unconstrained while
  scheduled runs are one-per-`(schedule, tick)`.

`TryInsertScheduledDelivery` is the conflict-guarded insert — `ON CONFLICT (schedule_id,
scheduled_tick) WHERE scheduled_tick IS NOT NULL DO NOTHING RETURNING …`, so a claimed tick
returns `pgx.ErrNoRows` and the dispatcher records the skip and commits, rendering nothing. The
dispatcher structure mirrors `queue.Dispatcher` exactly: a minute ticker, a per-schedule
`pg_advisory_xact_lock` serialising concurrent polls, and the unique key as the durable backstop.

### 3. In-instance only — generated, never delivered

A dispatched run's receipt is stamped `state = 'generated'` with `delivered_at` NULL. The
dispatcher cuts the artifact with the canonical renderer (`message.RenderArtifact`, result
discarded — confirming the report is cuttable for the window; content wiring lands later) but
sends nothing off-instance. Off-instance delivery is a separate ticket
([#508/T7](https://github.com/winniel123/verge-asm/issues/508)); [ADR-0039](./0039-a-channel-carries-the-message-never-the-estate-and-a-delivery-is-an-operational-record.md)
stands — a report run is neither the world moving nor our looking changing, it never becomes a
Message, and the receipt snapshots nothing (the artifact recomputes from the period bounds at
render time).

## Consequences

- **One function decides every window.** Run-now and the dispatcher call `report.CadenceWindow`,
  so a manual and a scheduled run of a schedule cover the same period by construction — they
  cannot drift.
- **No new dispatch table, no new join.** The receipt store carries one nullable column and one
  partial index. Reads that ignore `scheduled_tick` (the "Recurring reports" last-sent cell, the
  artifact view) are unchanged; `InsertReportDelivery` (Run-now) is unchanged and simply leaves
  the new column NULL.
- **Manual and scheduled runs coexist without collision.** The partial index constrains only
  tick-bearing rows, so a manual Run-now and an on-cadence run in the same window are two
  distinct receipts, as intended.
- **Missed ticks are not caught up.** The floored tick is always the current window, never a past
  one — a worker that was down over a window does not backfill it. This matches the queue
  dispatcher and is the same deliberate choice: currency, not history.
- **Custom cadences are coarse.** A cron string dispatches weekly. If minute-accurate custom
  cadence is ever required, it reopens this ADR and the ADR-0091 question — a real predicate
  evaluator is the thing ADR-0091 governs, and would be adopted, if ever, on the record.

## Alternatives rejected

| Alternative | Why not |
| --- | --- |
| Evaluate the operator's cron expression to fire the schedule | An operator-authored predicate over a versioned rule set — precisely what ADR-0091 forbids; and it needs a cron evaluator the tree does not have |
| A separate `report_dispatch` table keyed `(schedule, tick)` | Redundant: the receipt already IS the one-run record; a second table would duplicate the schedule/period/sequence it already holds |
| A non-partial unique on `(schedule_id, scheduled_tick)` | Would constrain manual Run-now receipts (all NULL tick) against each other — a NULL is distinct under a plain unique in Postgres, but the partial index states the intent exactly and keeps manual runs out of the on-cadence index |
| Deliver on dispatch (state `delivered`) | Off-instance send is #508/T7; ADR-0039 keeps every artifact in-instance until then — a `delivered` stamp with no send would lie |
