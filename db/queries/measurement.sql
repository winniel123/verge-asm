-- name: ListEnabledScans :many
SELECT id, kind, enabled, cadence_seconds, created_at
FROM scan
WHERE enabled = TRUE
ORDER BY id;

-- name: GetScanByKind :one
SELECT id, kind, enabled, cadence_seconds, created_at
FROM scan
WHERE kind = $1;

-- name: ListVantagesForDispatch :many
-- The dns Scan dispatches over every configured Vantage, reading only its
-- measurement identity (name, class, resolver). Distinct from the web prober
-- list (vantages.sql `ListVantages`), which is scoped to provisioned probers.
SELECT id, name, class, resolver, created_at
FROM vantage
ORDER BY id;

-- name: ListNameSeedDomains :many
SELECT name_domain
FROM seed
WHERE kind = 'name' AND name_domain IS NOT NULL
ORDER BY name_domain;

-- name: TryFanOut :one
-- Idempotent on (scan, scheduled_time): the first tick inserts a fanned-out
-- Dispatch; an overlapping tick conflicts and returns no row, which the caller
-- records as a skip rather than a second fan-out.
INSERT INTO dispatch (scan_id, scheduled_time, status)
VALUES ($1, $2, 'fanned-out')
ON CONFLICT ON CONSTRAINT dispatch_tick_key DO NOTHING
RETURNING id;

-- name: EnqueueJob :one
INSERT INTO queue_job (
    scan_id, vantage_id, dispatch_id, kind, spec, attempted_scope, offers,
    attempt, max_attempts, run_after
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: ClaimJob :one
-- The Postgres-backed claim: FOR UPDATE SKIP LOCKED over ready jobs whose
-- run_after has passed, oldest first, marking the winner running in one
-- statement so two workers never claim the same job.
UPDATE queue_job SET state = 'running'
WHERE id = (
    SELECT id FROM queue_job
    WHERE state = 'ready' AND run_after <= now()
    ORDER BY run_after, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, scan_id, vantage_id, dispatch_id, kind, spec, attempted_scope,
          offers, attempt, max_attempts;

-- name: InsertBatch :one
INSERT INTO batch (
    scan_id, dispatch_id, vantage_id, kind, outcome, offers, recorded_scope
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: InsertObservation :exec
INSERT INTO observation (
    batch_id, facet, subject_kind, subject_key, discriminator, vantage_id,
    source, value, observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: MarkJobDone :exec
UPDATE queue_job SET state = 'done', batch_id = $2 WHERE id = $1;

-- name: MarkJobDead :exec
UPDATE queue_job SET state = 'dead', batch_id = $2 WHERE id = $1;

-- name: MarkJobRetried :exec
UPDATE queue_job SET state = 'retried' WHERE id = $1;

-- name: CountObservationsForScan :one
SELECT count(*)
FROM observation o
JOIN batch b ON b.id = o.batch_id
WHERE b.scan_id = $1;

-- name: ListRecentObservations :many
SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
       o.source, o.value, o.observed_at, o.batch_id
FROM observation o
ORDER BY o.id DESC
LIMIT $1;
