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

-- name: DeleteExpiredDispatches :execrows
-- The one and only path that deletes Dispatch rows (v1 spec §4.6, ADR-0041). It
-- touches the dispatch table and nothing else: no Observation, Span, Batch or
-- queue_job row is read or written here, so retiring Dispatch can move no value
-- on any timeline. The FK change in migration 20900 lets the delete null the
-- operational back-references rather than cascade into measured data.
DELETE FROM dispatch
WHERE scheduled_time < $1;
