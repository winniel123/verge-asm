-- name: ListEnabledScans :many
SELECT id, kind, enabled, cadence_seconds, created_at
FROM scan
WHERE enabled = TRUE
ORDER BY id;

-- name: ListScans :many
SELECT id, kind, enabled, cadence_seconds, created_at
FROM scan
ORDER BY id;

-- name: GetScanByKind :one
SELECT id, kind, enabled, cadence_seconds, created_at
FROM scan
WHERE kind = $1;

-- name: ListVantagesForDispatch :many
SELECT id, name, class, resolver, egress, dialled_addr, created_at
FROM vantage
ORDER BY id;

-- name: ListNameSeedDomains :many
SELECT name_domain
FROM seed
WHERE kind = 'name' AND name_domain IS NOT NULL
ORDER BY name_domain;

-- name: ListAddressScopeCidrs :many
SELECT address_cidr
FROM seed
WHERE kind = 'address' AND address_cidr IS NOT NULL
ORDER BY id;

-- name: ListExtendedZoneDomains :many
SELECT name_domain
FROM seed
WHERE kind = 'name' AND custody_extension = TRUE AND name_domain IS NOT NULL
ORDER BY name_domain;

-- name: ListVergeCoreFrequencyEdits :many
SELECT port, action
FROM verge_core_frequency_edit
ORDER BY id;

-- name: TryFanOut :one
INSERT INTO dispatch (scan_id, scheduled_time, status)
VALUES ($1, $2, 'fanned-out')
ON CONFLICT ON CONSTRAINT dispatch_tick_key DO NOTHING
RETURNING id;

-- name: ScanHasNonTerminalJobs :one
SELECT EXISTS (
    SELECT 1
    FROM queue_job
    WHERE scan_id = @scan_id::bigint
      -- The Dispatch sweep nulls it, so a plain <> would drop a job that must still hold the gate.
      AND dispatch_id IS DISTINCT FROM @dispatch_id::bigint
      AND state IN ('ready', 'running')
) AS lagging;

-- name: EnqueueJob :one
INSERT INTO queue_job (
    scan_id, vantage_id, dispatch_id, kind, spec, attempted_scope, offers,
    attempt, max_attempts, run_after
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id;

-- name: NotifyJobProgress :exec
SELECT pg_notify('queue_job_progress', @payload::text);

-- name: ClaimJob :one
UPDATE queue_job SET state = 'running', claimed_at = now()
WHERE id = (
    SELECT id FROM queue_job
    WHERE state = 'ready' AND run_after <= now()
    ORDER BY run_after, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, scan_id, vantage_id, dispatch_id, kind, spec, attempted_scope,
          offers, attempt, max_attempts;

-- name: ReapStaleRunningJobs :execrows
-- A dead worker is failure, not evidence, so a reap writes no Batch and moves no Availability (ADR-0169 §1, #1391).
UPDATE queue_job
SET state      = CASE WHEN attempt >= max_attempts THEN 'dead' ELSE 'ready' END,
    attempt    = attempt + 1,
    run_after  = now(),
    claimed_at = NULL
WHERE state = 'running' AND claimed_at < @cutoff::timestamptz;

-- name: InsertBatch :one
INSERT INTO batch (
    scan_id, dispatch_id, vantage_id, kind, outcome, offers, recorded_scope
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: PreviousBatchTime :one
SELECT max(created_at)::timestamptz AS prev_batch_at
FROM batch
WHERE created_at < (SELECT max(created_at) FROM batch);

-- name: EarliestBatchTime :one
SELECT min(created_at)::timestamptz AS earliest_batch_at
FROM batch;

-- name: InsertObservation :exec
INSERT INTO observation (
    batch_id, facet, subject_kind, subject_key, discriminator, vantage_id,
    source, value, observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: MarkJobDone :execrows
UPDATE queue_job SET state = 'done', batch_id = $2 WHERE id = $1 AND state = 'running';

-- name: MarkJobDead :execrows
UPDATE queue_job SET state = 'dead', batch_id = $2 WHERE id = $1 AND state = 'running';

-- name: MarkJobRetried :execrows
UPDATE queue_job SET state = 'retried' WHERE id = $1 AND state = 'running';

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

-- name: NameCitedAddresses :many
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key, o.vantage_id)
        o.subject_key AS subject_key,
        o.value->>'outcome' AS outcome,
        o.value AS value
    FROM live o
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.vantage_id, o.observed_at DESC
)
SELECT DISTINCT
    subject_key,
    jsonb_array_elements_text(value->'addresses') AS address
FROM latest
WHERE outcome = 'Resolved'
ORDER BY subject_key, address;

-- name: ScanHasCompletedBatch :one
SELECT EXISTS (
    SELECT 1 FROM batch WHERE kind = $1 AND outcome = 'completed'
) AS completed;
