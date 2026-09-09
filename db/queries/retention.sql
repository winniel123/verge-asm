-- name: GetRetentionSettings :one
-- The migration seeds the one global row, so a missing row is a fault and not an empty state.
SELECT observation_currency_days, dispatch_cadence_multiple,
       transcript_currency_days, updated_by, updated_at
FROM retention_settings
WHERE id = true;

-- name: UpdateRetentionSettings :exec
UPDATE retention_settings
SET observation_currency_days = $1, dispatch_cadence_multiple = $2,
    transcript_currency_days = $3, updated_by = $4, updated_at = now()
WHERE id = true;

-- name: SlowestEnabledScanCadenceSeconds :one
SELECT COALESCE(MAX(cadence_seconds), 0)::bigint AS cadence_seconds
FROM scan
WHERE enabled = TRUE;

-- name: TightestEnabledScanCadenceSeconds :one
SELECT COALESCE(MIN(cadence_seconds), 0)::bigint AS cadence_seconds
FROM scan
WHERE enabled = TRUE;

-- name: ListCoveringScanKinds :many
-- Cover is the cover CTE's relation, as a semi-join that stops at the first row (#1768).
SELECT s.kind
FROM scan s
WHERE s.enabled = TRUE
  AND EXISTS (
      SELECT 1
      FROM batch b
      JOIN observation o ON o.batch_id = b.id
      WHERE b.scan_id = s.id
  )
ORDER BY s.kind;

-- name: ListLiveObservationsForDerivation :many
-- Every derivation read of observation inlines this gate, never the raw table (#237, ADR-0041).
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
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
withdrawn AS (
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
            w.subject_key IS NOT NULL
         OR (c.tightest_cadence IS NOT NULL
             AND EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
                 > sqlc.arg(floor_cadences)::bigint * c.tightest_cadence)
          )
);

-- name: DeleteExpiredDispatches :execrows
DELETE FROM dispatch
WHERE scheduled_time < $1;

-- name: DeleteExpiredTranscripts :execrows
DELETE FROM transcript
WHERE captured_at < $1;

-- name: CountHeldObservations :one
-- The corpus reaches ~98M rows a year at the ceiling, so the count caps (ADR-0081, #1768).
SELECT
    (SELECT COUNT(*)::bigint
       FROM (SELECT 1 FROM observation LIMIT sqlc.arg(exact_limit)::bigint + 1) capped
    )::bigint AS counted_rows,
    GREATEST(pg_stat_get_live_tuples('observation'::regclass), 0)::bigint AS estimated_rows;

-- name: ListFacetSourceFloors :many
-- The pair's floor is the tightest bound in force across the pair, reached through each
-- row's Batch as the retirement query reaches it. A row from a disabled Scan has no bound
-- and is counted apart, because the sweep never retires it.
SELECT o.facet,
       o.source,
       COALESCE(MIN(s.cadence_seconds), 0)::bigint AS tightest_cadence,
       COALESCE(
           (ARRAY_AGG(s.kind ORDER BY s.cadence_seconds, s.kind)
            FILTER (WHERE s.kind IS NOT NULL))[1],
           ''
       )::text AS scan_kind,
       COUNT(*)::bigint AS rows_held,
       COUNT(*) FILTER (WHERE s.id IS NULL)::bigint AS uncovered_rows
FROM observation o
JOIN batch b ON b.id = o.batch_id
LEFT JOIN scan s ON s.id = b.scan_id AND s.enabled = TRUE
GROUP BY o.facet, o.source
ORDER BY o.facet, o.source;

-- name: ListDerivationBreaks :many
-- A Break is derived on read from two adjacent spans' vectors and never stored, so the
-- moved leaf is named by diffing the pair in Go.
WITH adjacent AS (
    SELECT opened_at,
           derivation,
           lag(derivation) OVER (
               PARTITION BY subject_key, facet, discriminator, vantage_id, source
               ORDER BY opened_at
           ) AS previous
    FROM span
)
SELECT opened_at, derivation, previous::jsonb AS previous
FROM adjacent
WHERE previous IS NOT NULL
  AND derivation <> previous
ORDER BY opened_at DESC
LIMIT sqlc.arg(row_limit)::bigint;
