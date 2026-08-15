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
