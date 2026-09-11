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

-- name: GetDnsCadenceSeconds :one
SELECT cadence_seconds FROM scan WHERE kind = 'dns';

-- name: SetDnsCadenceSeconds :exec
-- A non-positive interval is refused by the table's CHECK, not by this statement.
UPDATE scan SET cadence_seconds = $1 WHERE kind = 'dns';

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
-- A dead worker is failure, not evidence: no Batch, no Availability move (ADR-0169 §1, #1391).
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

-- name: FoldedBatchWindow :one
WITH done AS (
    -- A dead-lettered or CT or zone batch folds no message, so it may not bound the window (#1728).
    SELECT created_at FROM batch
    WHERE outcome = 'completed' AND kind <> ALL(sqlc.arg(unfolded_kinds)::text[])
)
SELECT (SELECT max(created_at) FROM done
        WHERE created_at < (SELECT max(created_at) FROM done))::timestamptz AS prev_at,
       (SELECT max(created_at) FROM done)::timestamptz AS latest_at;

-- name: EarliestBatchTime :one
SELECT min(created_at)::timestamptz AS earliest_batch_at
FROM batch;

-- name: InsertObservation :exec
INSERT INTO observation (
    batch_id, facet, subject_kind, subject_key, discriminator, vantage_id,
    source, value, observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: RenewJobLease :execrows
-- A ct-tail job outlives the stale threshold, so the owner renews off any transaction (#1709).
UPDATE queue_job SET claimed_at = now() WHERE id = $1 AND state = 'running';

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
        o.vantage_id AS vantage_id,
        o.batch_id AS batch_id,
        o.value->>'outcome' AS outcome,
        o.value AS value
    FROM live o
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.vantage_id, o.observed_at DESC
),
cited AS (
    SELECT l.subject_key, l.vantage_id, l.batch_id,
           jsonb_array_elements_text(l.value->'addresses') AS address
    FROM latest l
    WHERE l.outcome = 'Resolved'
),
-- The owner rides the batch's dns-record rows, so no leaf version moves (ADR-0151 §2, #1678).
terminal AS (
    SELECT o.subject_key, o.vantage_id, o.batch_id,
           rr->>'data' AS address,
           rr->>'name' AS owner
    FROM live o
    CROSS JOIN LATERAL jsonb_array_elements(o.value->'rrs') AS rr
    WHERE o.facet = 'dns-record' AND o.subject_kind = 'name'
      AND rr->>'type' IN ('A', 'AAAA')
)
SELECT DISTINCT
    c.subject_key,
    c.address,
    COALESCE(t.owner, c.subject_key)::text AS owner
FROM cited c
LEFT JOIN terminal t
    ON  t.subject_key = c.subject_key
    AND t.vantage_id IS NOT DISTINCT FROM c.vantage_id
    AND t.batch_id   = c.batch_id
    AND t.address    = c.address
ORDER BY c.subject_key, c.address, owner;

-- name: ScanHasCompletedBatch :one
SELECT EXISTS (
    SELECT 1 FROM batch WHERE kind = $1 AND outcome = 'completed'
) AS completed;

-- name: ReachFoldedBeforeAtVantages :one
-- EXISTS stops at the first hit, so the kind range is read in full once per class (#1731).
SELECT EXISTS (
    SELECT 1 FROM batch
    WHERE kind = ANY(sqlc.arg(kinds)::text[])
      AND outcome = 'completed'
      AND vantage_id = ANY(sqlc.arg(vantage_ids)::bigint[])
      AND id < sqlc.arg(before_batch_id)::bigint
) AS folded;

-- name: ListSettleableRePointBatches :many
-- The arm below is ListReleasableHeldMessages', so both paths read one tier (ADR-1806 §3).
SELECT b.id,
       EXISTS (
           SELECT 1
           FROM span n
           JOIN span p
             ON p.subject_key = n.subject_key
            AND p.facet = n.facet
            AND p.discriminator = n.discriminator
            AND p.vantage_id IS NOT DISTINCT FROM n.vantage_id
            AND p.source = n.source
            AND p.closed_batch_id = n.opened_batch_id
           WHERE n.opened_batch_id = b.id
             AND n.subject_kind = 'name'
             AND n.facet = 'resolution'
             AND n.is_gap = FALSE
             AND p.is_gap = FALSE
       ) AS has_move
FROM batch b
WHERE b.repoint_settled_at IS NULL
  AND (
      -- No reaper leaves the drain test reading a job set nothing reaps (ADR-1806 §6).
      sqlc.arg(reaper_disabled)::boolean
      -- A disabled hot tier opens nothing beneath the move, ever (ADR-1806 §6).
      OR NOT EXISTS (
          SELECT 1 FROM scan hs WHERE hs.kind = 'hot' AND hs.enabled
      )
      OR EXISTS (
          SELECT 1
          FROM (
              SELECT d.id
              FROM dispatch d
              JOIN scan s ON s.id = d.scan_id
              WHERE s.kind = 'hot'
                -- A skipped tick enqueues no job and reads as drained (ADR-1806 §2).
                AND d.status = 'fanned-out'
                AND d.created_at >= b.created_at
              ORDER BY d.created_at, d.id
              LIMIT 1
          ) first_hot
            -- An empty job set is a mid-fan-out dispatch and not a drained one (#1816).
          WHERE EXISTS (
              SELECT 1 FROM queue_job j
              WHERE j.dispatch_id = first_hot.id
          )
          AND NOT EXISTS (
              -- The cadence-lag gate's query excludes this dispatch's own jobs (ADR-1806 §3).
              SELECT 1 FROM queue_job j
              WHERE j.dispatch_id = first_hot.id
                AND j.state IN ('ready', 'running')
          )
      )
  )
ORDER BY b.id;

-- name: SettleRePointBatch :execrows
-- The guarded UPDATE is the claim, so one fold announces its moves once (ADR-1806 §8, #1818).
UPDATE batch
SET repoint_settled_at = sqlc.arg(settled_at)
WHERE id = sqlc.arg(id) AND repoint_settled_at IS NULL;

-- name: SettleMovelessRePointBatches :exec
-- A fold that moved no resolution owes no message, so it needs no pass of its own (#1818).
UPDATE batch
SET repoint_settled_at = sqlc.arg(settled_at)
WHERE id = ANY(sqlc.arg(ids)::bigint[]) AND repoint_settled_at IS NULL;
