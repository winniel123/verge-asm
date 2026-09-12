-- name: ListDispatchProgress :many
SELECT
    d.id       AS dispatch_id,
    d.scan_id  AS scan_id,
    s.kind     AS scan_kind,
    d.created_at,
    d.status   AS status,
    count(j.id)                                 AS total,
    count(*) FILTER (WHERE j.state = 'ready')   AS ready,
    count(*) FILTER (WHERE j.state = 'running') AS running,
    count(*) FILTER (WHERE j.state = 'done')    AS done,
    count(*) FILTER (WHERE j.state = 'dead')    AS dead,
    count(*) FILTER (WHERE j.state = 'retried') AS retried
FROM dispatch d
JOIN scan s ON s.id = d.scan_id
LEFT JOIN queue_job j ON j.dispatch_id = d.id
GROUP BY d.id, d.scan_id, s.kind, d.created_at, d.status
ORDER BY d.id DESC
LIMIT $1;

-- name: ListActiveDispatchProgress :many
SELECT
    d.id       AS dispatch_id,
    d.scan_id  AS scan_id,
    s.kind     AS scan_kind,
    d.created_at,
    d.status   AS status,
    count(j.id)                                 AS total,
    count(*) FILTER (WHERE j.state = 'ready')   AS ready,
    count(*) FILTER (WHERE j.state = 'running') AS running,
    count(*) FILTER (WHERE j.state = 'done')    AS done,
    count(*) FILTER (WHERE j.state = 'dead')    AS dead,
    count(*) FILTER (WHERE j.state = 'retried') AS retried
FROM dispatch d
JOIN scan s ON s.id = d.scan_id
LEFT JOIN queue_job j ON j.dispatch_id = d.id
GROUP BY d.id, d.scan_id, s.kind, d.created_at, d.status
HAVING count(*) FILTER (WHERE j.state IN ('ready', 'running')) > 0
ORDER BY d.id DESC;

-- name: ListConcludedDispatchProgress :many
SELECT
    d.id       AS dispatch_id,
    d.scan_id  AS scan_id,
    s.kind     AS scan_kind,
    d.created_at,
    d.status   AS status,
    count(j.id)                                 AS total,
    count(*) FILTER (WHERE j.state = 'ready')   AS ready,
    count(*) FILTER (WHERE j.state = 'running') AS running,
    count(*) FILTER (WHERE j.state = 'done')    AS done,
    count(*) FILTER (WHERE j.state = 'dead')    AS dead,
    count(*) FILTER (WHERE j.state = 'retried') AS retried
FROM dispatch d
JOIN scan s ON s.id = d.scan_id
LEFT JOIN queue_job j ON j.dispatch_id = d.id
GROUP BY d.id, d.scan_id, s.kind, d.created_at, d.status
HAVING count(*) FILTER (WHERE j.state IN ('ready', 'running')) = 0
ORDER BY d.id DESC
LIMIT $1;

-- name: CancelReadyJobsForDispatch :execrows
UPDATE queue_job SET state = 'cancelled'
WHERE dispatch_id = $1 AND state = 'ready';

-- name: CancelActiveJobsForDispatch :execrows
UPDATE queue_job SET state = 'cancelled'
WHERE dispatch_id = $1 AND state IN ('ready', 'running');

-- name: SetDispatchStatus :exec
UPDATE dispatch SET status = $2
WHERE id = $1
  AND (status = 'fanned-out' OR (status = 'stopped' AND $2 = 'terminated'));

-- name: ListJobsForDispatch :many
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

-- name: MarkFanOutComplete :exec
-- The release bound reads this as its fan-out-finished half (ADR-1806 §3, ADR-1851 §2).
UPDATE dispatch SET fanout_complete = true
WHERE id = $1;

-- name: AbandonUnfinishedDispatches :execrows
-- A crashed fan-out marks itself never, so the next claimed tick retires it (ADR-1851 §3).
UPDATE dispatch SET fanout_abandoned = true
WHERE dispatch.scan_id = $1
  AND dispatch.id <> sqlc.arg(claimed_id)
  AND dispatch.status = 'fanned-out'
  AND dispatch.fanout_complete = false
  AND dispatch.fanout_abandoned = false
  -- A fan-out still streaming holds ready jobs, so this leaves a live one alone (ADR-1851 §3).
  AND NOT EXISTS (
      SELECT 1 FROM queue_job j
      WHERE j.dispatch_id = dispatch.id
        AND j.state IN ('ready', 'running')
  );
