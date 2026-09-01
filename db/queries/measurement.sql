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
-- The dns Scan dispatches over every configured Vantage, reading its measurement
-- identity (name, resolver) and its presented-address facts (egress + dialled_addr),
-- from which the hot/cold Scans DERIVE its class per batch for the Custody gate — never
-- the vestigial `class` column (#709, ADR-0079). Distinct from the web prober list
-- (vantages.sql `ListVantages`), which is scoped to provisioned probers.
SELECT id, name, class, resolver, egress, dialled_addr, created_at
FROM vantage
ORDER BY id;

-- name: ListNameSeedDomains :many
SELECT name_domain
FROM seed
WHERE kind = 'name' AND name_domain IS NOT NULL
ORDER BY name_domain;

-- name: ListAddressScopeCidrs :many
-- The declared address-scope Seeds, for the hot Scan's Custody derivation: every
-- address inside one derives operator directly (ADR-0013).
SELECT address_cidr
FROM seed
WHERE kind = 'address' AND address_cidr IS NOT NULL
ORDER BY id;

-- name: ListExtendedZoneDomains :many
-- The registrable domains of custody-extended name-scope Seeds, for the hot
-- Scan's Custody derivation: an address a name in one of these zones resolves to
-- derives operator by extension (ADR-0013 §3).
SELECT name_domain
FROM seed
WHERE kind = 'name' AND custody_extension = TRUE AND name_domain IS NOT NULL
ORDER BY name_domain;

-- name: ListVergeCoreFrequencyEdits :many
-- The operator's edits to verge-core's frequency half (v1 spec §3.5). Only the
-- frequency half is operator-editable; these deltas are applied over the shipped
-- default at hot fan-out.
SELECT port, action
FROM verge_core_frequency_edit
ORDER BY id;

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

-- name: NotifyJobProgress :exec
-- Publish one ephemeral, redacted per-job progress event over the
-- queue_job_progress LISTEN/NOTIFY channel (#780, collision #40 producer half).
-- The payload is a small JSON line the RunDetail live stream enriches its
-- state-derived log with while a job is in flight. NOTHING is persisted at rest:
-- pg_notify delivers the payload to connected listeners and is gone (ADR-0041's
-- corpus separation and the instance-privacy posture are untouched — there is no
-- raw-stdout column or table). Fired inside the job's terminal transaction, so a
-- job cancelled mid-flight rolls its event back with the rest of its work.
SELECT pg_notify('queue_job_progress', @payload::text);

-- name: ClaimJob :one
-- The Postgres-backed claim: FOR UPDATE SKIP LOCKED over ready jobs whose
-- run_after has passed, oldest first, marking the winner running in one
-- statement so two workers never claim the same job. It stamps claimed_at at the
-- claim instant so the stale-running reaper (internal/queue/reaper.go, #853) knows
-- when the lease started and can reclaim a job whose worker died or hung mid-run.
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
-- The stale-`running` reaper's sweep (#853): reclaim every job stuck in state
-- 'running' whose lease (claimed_at) is older than the cutoff — the worker that
-- claimed it died or hung mid-job, so nothing will ever drive it to a terminal
-- state. A job with attempts left returns to 'ready' (attempt bumped, run_after and
-- claimed_at cleared) so a live worker re-claims and re-runs it — a fresh Batch, not
-- a resumption, since a job orphaned mid-run committed no Batch (batch_id is NULL).
-- A job past its attempt budget is dead-lettered directly, which bounds a job whose
-- prober hangs on every attempt: it dies after max_attempts reaps rather than
-- re-readying forever.
--
-- The CASE reads the OLD attempt, so a job at attempt >= max_attempts dies and one
-- below it re-readies at attempt + 1. Only 'running' rows past the cutoff match; a
-- NULL claimed_at never satisfies `< cutoff`, so a never-leased row is never reaped.
-- The reaper writes no Batch and moves no Availability: a dead worker is
-- infrastructure failure, not measurement evidence, so a reaped dns job must not be
-- read as a resolver outage (ADR-0108). Returns the count reclaimed.
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
-- The commit instant of the second-most-recent distinct batch — the boundary a
-- vs-last-batch stat delta reads the "value a batch ago" at (P0.2). It is the most
-- recent batch instant strictly before the latest, so the span population open at
-- it is the estate exactly as the previous batch left it, with only the most recent
-- batch's opens and closes lying between it and now. NULL where fewer than two
-- distinct batch instants exist — the first batch has no predecessor to compare
-- against, so a delta is withheld rather than compared against nothing. Reads batch
-- only (corpus 1), never dispatch, honoring the comparison-path separation (ADR-0041).
SELECT max(created_at)::timestamptz AS prev_batch_at
FROM batch
WHERE created_at < (SELECT max(created_at) FROM batch);

-- name: EarliestBatchTime :one
-- The commit instant of the FIRST batch the estate ever folded — the age boundary the
-- Drift page's vs-previous-period delta tests before comparing (P0.12, #690). The chip
-- compares the selected window against the immediately preceding equal-length window;
-- that comparison is only honest once the estate has been observing since at or before
-- the preceding window's start, so the delta is suppressed while the earliest batch is
-- younger than that (install younger than 2× the window), never a fabricated baseline.
-- NULL where no batch has committed. Reads batch only (corpus 1), never dispatch (ADR-0041).
SELECT min(created_at)::timestamptz AS earliest_batch_at
FROM batch;

-- name: InsertObservation :exec
INSERT INTO observation (
    batch_id, facet, subject_kind, subject_key, discriminator, vantage_id,
    source, value, observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: MarkJobDone :execrows
-- Guarded on the job still being 'running': a terminate (DF-F4) that cancelled the
-- job mid-flight left it 'cancelled', so this affects no row and the caller rolls the
-- transaction back — the staged batch and observations are discarded (job atomicity,
-- worker.go). A job the worker owns uncontested is 'running', so the update lands and
-- returns 1.
UPDATE queue_job SET state = 'done', batch_id = $2 WHERE id = $1 AND state = 'running';

-- name: MarkJobDead :execrows
-- Guarded on 'running' exactly as MarkJobDone — a job a terminate cancelled mid-flight
-- does not dead-letter; its transaction rolls back and its work is discarded.
UPDATE queue_job SET state = 'dead', batch_id = $2 WHERE id = $1 AND state = 'running';

-- name: MarkJobRetried :execrows
-- Guarded on 'running': a job a terminate cancelled mid-flight is not retried, so the
-- fresh attempt is never enqueued (the caller rolls back on a zero count).
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
-- The Addresses a current resolution cites, per Name — an `Address` is in the
-- estate exactly while a current resolution cites it. Only a `Resolved` value
-- cites; a `Shadowed` (or NoData / NameError / Lame / Gap) value cites nothing,
-- so every `Address` held only by a superseded `Resolved` leaves the estate.
-- Reads through the live-tier gate (#237, ADR-0041): the hot Scan's Custody
-- derivation admits an Address only while a resolution a derivation may still read
-- cites it, so the `cover`/`live` CTE pair below (the inlined twin of
-- ListLiveObservationsForDerivation, evaluated at @as_of with k = @floor_cadences)
-- keeps an Address held only by an evidential answer out of the probed estate.
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
-- Whether a `Scan` of this kind has ever completed a Batch. A Batch row exists only at
-- a terminal outcome, so this asks whether the Scan has actually RUN on this install,
-- as against being merely enabled.
--
-- The `edge-fanout` veto reads it to tell its two empty states apart (#985, ADR-0129
-- §4): a Scan that has not run yet, whose candidates are *measurement pending* and are
-- HELD, from a Scan that runs and records nothing, which is an ERRORED Scan and opens
-- the reach. A dead-lettered Batch does not count — it is the job failing, and the tick
-- retries.
SELECT EXISTS (
    SELECT 1 FROM batch WHERE kind = $1 AND outcome = 'completed'
) AS completed;
