-- Reads and writes behind the report_delivery receipts store (#291/T2). A
-- report_delivery is the Operational record of one run of a report_schedule: it
-- has no cause and never becomes a Message (ADR-0039, ADR-0081). It backs the
-- "Recurring reports" table's "last sent" cell and the delivered-artifact view.
-- The receipt stores only the run's period bounds and outcome — the artifact
-- recomputes its contents from those bounds at render time, snapshotting nothing.

-- name: NextReportDeliveryNo :one
-- The next 1-based per-schedule sequence number for a run: max+1, or 1 for the
-- first. The caller passes it to InsertReportDelivery; the unique (schedule_id,
-- delivery_no) key keeps the sequence dense under a single writer.
SELECT COALESCE(MAX(delivery_no), 0) + 1 AS next_no
FROM report_delivery
WHERE schedule_id = $1;

-- name: InsertReportDelivery :one
-- Record one run of a schedule for a bounded period. delivery_no is the caller's
-- next-sequence read (NextReportDeliveryNo); state is one of generated / delivered
-- / failed; delivered_at is NULL where the run generated without leaving (a
-- download-only schedule) and the stamp otherwise. generated_at defaults to now().
INSERT INTO report_delivery (schedule_id, period_start, period_end, delivery_no, state, delivered_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick;

-- name: TryInsertScheduledDelivery :one
-- Claim one on-cadence run of a schedule for a tick, idempotently: the partial
-- unique (schedule_id, scheduled_tick) admits only the first poll in a window; a
-- later poll conflicts and returns no row (a recorded skip, not a double-run),
-- mirroring the queue dispatcher's TryFanOut. delivery_no is the caller's
-- NextReportDeliveryNo read; state is 'generated' and delivered_at NULL — an
-- in-instance run generates without leaving (off-instance send is T7/#508, blocked).
INSERT INTO report_delivery (schedule_id, period_start, period_end, delivery_no, state, delivered_at, scheduled_tick)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (schedule_id, scheduled_tick) WHERE scheduled_tick IS NOT NULL DO NOTHING
RETURNING id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state;

-- name: GetLatestReportDelivery :one
-- The newest non-failed run of a schedule — the receipt the "Recurring reports"
-- table reads for its "last sent" cell and the artifact view opens. A failed run
-- is not a delivery to view, so it is excluded; where a schedule has never run (or
-- only failed) this returns no row and the caller renders the em-dash empty-state
-- rather than fabricating a delivery (ADR-0110).
SELECT id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick
FROM report_delivery
WHERE schedule_id = $1 AND state <> 'failed'
ORDER BY id DESC
LIMIT 1;

-- name: ListReportDeliveries :many
-- Every run of one schedule, newest-first — the delivery history behind a
-- schedule, including failed runs so the record is complete.
SELECT id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick
FROM report_delivery
WHERE schedule_id = $1
ORDER BY id DESC;
