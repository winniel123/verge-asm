-- name: GetRetentionSettings :one
-- The single operator-global row seeded by the migration; it always exists.
SELECT observation_currency_days, dispatch_cadence_multiple, updated_by, updated_at
FROM retention_settings
WHERE id = true;

-- name: UpdateRetentionSettings :exec
UPDATE retention_settings
SET observation_currency_days = $1, dispatch_cadence_multiple = $2,
    updated_by = $3, updated_at = now()
WHERE id = true;

-- name: SlowestEnabledScanCadenceSeconds :one
-- The slowest enabled Scan's cadence — the largest cadence_seconds among enabled
-- Scans — which the Dispatch dial is a multiple of and the floor k multiples of
-- (v1 spec §4.6). COALESCE to 0 when no Scan is enabled: with no cadence the
-- multiple has no meaning, so the sweep treats it as unbounded and retires
-- nothing. It reads only the scan table and never the operational or measured
-- corpora.
SELECT COALESCE(MAX(cadence_seconds), 0)::bigint AS cadence_seconds
FROM scan
WHERE enabled = TRUE;

-- name: TightestEnabledScanCadenceSeconds :one
-- The tightest enabled Scan's cadence — the smallest cadence_seconds among enabled
-- Scans — which is k cadences of the smallest per-timeline observation bound any
-- in-force timeline can carry, and therefore the observation dial's floor (v1 spec
-- §4.6, ADR-0094). Symmetric to the Dispatch floor's SlowestEnabledScanCadence
-- (which takes MAX): Dispatch floors at the slowest Scan, observations at the
-- tightest. COALESCE to 0 when no Scan is enabled: with no bound in force there is
-- nothing to floor against and the dial is unconstrained. Reads only the scan
-- table, never the measured corpora.
SELECT COALESCE(MIN(cadence_seconds), 0)::bigint AS cadence_seconds
FROM scan
WHERE enabled = TRUE;

-- name: ListLiveObservationsForDerivation :many
-- The read-side live-tier gate a derivation reads observations through (v1 spec
-- §4.6, ADR-0041): it returns ONLY live-tier rows — those within k cadences of the
-- tightest ENABLED Scan covering their timeline — so a derivation reading through
-- it cannot re-derive history from a stale observation. NOTE: in v1 the
-- live/evidential boundary is enforced by RETIREMENT (DeleteExpiredObservations
-- below is the sole delete path and removes evidential rows on the Retirer's
-- sweep); the existing derivation reads still query the observation table directly
-- and see only live rows once a sweep has run. Routing each of those reads through
-- this gate — threading @as_of into every derivation query — is the stronger,
-- immediate separation and is the remaining integration. The bound is evaluated
-- PER TIMELINE and never collapsed:
-- `cover` groups by the full timeline key (subject, facet, discriminator, vantage,
-- source) and takes the tightest covering cadence, so a zone-sourced row's live
-- window is the zone cadence and a resolver-sourced row's is the resolver's. A
-- timeline no enabled Scan covers has an undefined bound: the INNER JOIN drops it,
-- so it yields no live row (it is retained as evidence, not read). @floor_cadences
-- is k; @as_of is the read instant.
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
)
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
ORDER BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source, o.observed_at;

-- name: DeleteExpiredObservations :execrows
-- Retire the evidential observations the operator's dial no longer keeps (v1 spec
-- §4.6, ADR-0041, ADR-0094). This is the ONLY path that deletes Observation rows,
-- and it deletes Observation rows and NOTHING else: batch, scan and span are read
-- to resolve each row's own bound and its subject's membership, never written — so
-- a Batch travels with any observation it produced (it is not retired per row) and
-- a Span is never compacted.
--
-- The query evaluates EACH ROW'S OWN bound, never a collapsed one: `cover` groups
-- observations by their full timeline key (subject, facet, discriminator, vantage,
-- source) and takes the tightest ENABLED covering Scan's cadence, so a zone-sourced
-- row ages on the zone cadence and a resolver-sourced row on the resolver's. A row
-- survives while its age is inside EITHER k cadences of that bound OR the dial,
-- whichever is longer — the control collapses to one number, the query never does.
-- Two populations fall opposite ways: a timeline no enabled Scan covers has an
-- undefined bound and is NEVER retired (the `cover` LEFT JOIN misses, so the guard
-- excludes it); a withdrawn subject (every span closed) carries NO floor, so the
-- dial alone governs it.
--
-- @dial_seconds is the operator's observation dial in seconds (0 == unbounded, the
-- v1 default — the caller does not run the sweep then); @floor_cadences is k; @as_of
-- is the sweep instant, injected so a sweep is reproducible.
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
withdrawn AS (
    -- A withdrawn subject's timelines are all closed: it has spans and no open one.
    SELECT subject_key
    FROM span
    GROUP BY subject_key
    HAVING bool_and(closed_at IS NOT NULL)
)
DELETE FROM observation obs
WHERE obs.id IN (
    SELECT o.id
    FROM observation o
    LEFT JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    LEFT JOIN withdrawn w ON w.subject_key = o.subject_key
    WHERE sqlc.arg(dial_seconds)::bigint > 0
      AND EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at)) > sqlc.arg(dial_seconds)::bigint
      AND (
            -- Withdrawn: no floor, the dial alone governs (the guard above is the
            -- whole test).
            w.subject_key IS NOT NULL
            -- Defined bound, not withdrawn: also past its own live bound. A row
            -- inside k cadences of its covering Scan is live and never reached here.
         OR (c.tightest_cadence IS NOT NULL
             AND EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
                 > sqlc.arg(floor_cadences)::bigint * c.tightest_cadence)
            -- c.tightest_cadence IS NULL and not withdrawn => undefined bound =>
            -- never retired (neither branch fires).
          )
);

-- name: DeleteExpiredDispatches :execrows
-- The one and only path that deletes Dispatch rows (v1 spec §4.6, ADR-0041). It
-- touches the dispatch table and nothing else: no Observation, Span, Batch or
-- queue_job row is read or written here, so retiring Dispatch can move no value
-- on any timeline. The FK change in migration 20900 lets the delete null the
-- operational back-references rather than cascade into measured data.
DELETE FROM dispatch
WHERE scheduled_time < $1;
