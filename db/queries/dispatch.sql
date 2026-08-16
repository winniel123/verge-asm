-- name: ListDispatchProgress :many
-- The recent Dispatches with their per-state job counts, newest first — the read
-- behind the Scans monitor (#245). Dispatch, queue_job and batch are Operational:
-- they record what the system did, never what is true of the estate, so this read
-- is barred from the comparison path by construction (CONTEXT.md, ADR-0041) and the
-- drift engine never sees it. A retry enqueues a fresh job and marks the old one
-- 'retried' (internal/queue/worker.go), so 'retried' rows are superseded attempts:
-- the live work is total − retried, of which done + dead are complete and
-- ready + running are still in flight. The LEFT JOIN keeps a Dispatch whose jobs
-- were retired to NULL by the Dispatch sweep (ADR-0041) — it counts zero jobs
-- rather than vanishing.
-- created_at is the instant the fan-out actually happened, which is what "when did
-- this scan start" means to an operator; scheduled_time is the cadence tick the
-- Dispatch is idempotent on, not a wall-clock start, so it is not read here.
SELECT
    d.id       AS dispatch_id,
    d.scan_id  AS scan_id,
    s.kind     AS scan_kind,
    d.created_at,
    count(j.id)                                 AS total,
    count(*) FILTER (WHERE j.state = 'ready')   AS ready,
    count(*) FILTER (WHERE j.state = 'running') AS running,
    count(*) FILTER (WHERE j.state = 'done')    AS done,
    count(*) FILTER (WHERE j.state = 'dead')    AS dead,
    count(*) FILTER (WHERE j.state = 'retried') AS retried
FROM dispatch d
JOIN scan s ON s.id = d.scan_id
LEFT JOIN queue_job j ON j.dispatch_id = d.id
GROUP BY d.id, d.scan_id, s.kind, d.created_at
ORDER BY d.id DESC
LIMIT $1;

-- name: ListJobsForDispatch :many
-- The per-job detail for one Dispatch — the progress drill-down (#245). Ordered by
-- id so a retried attempt reads immediately before the fresh job that replaced it.
-- A job's Batch outcome is NULL until the job reaches a terminal state, since a
-- Batch is written only at completion or dead-letter (db/migrations/18804); the
-- Vantage name is NULL for the zone Scan, which has no vantage choice at all.
SELECT
    j.id,
    j.kind,
    j.state,
    j.attempt,
    j.max_attempts,
    j.vantage_id,
    v.name    AS vantage_name,
    j.batch_id,
    b.outcome AS batch_outcome
FROM queue_job j
LEFT JOIN vantage v ON v.id = j.vantage_id
LEFT JOIN batch b ON b.id = j.batch_id
WHERE j.dispatch_id = $1
ORDER BY j.id;
