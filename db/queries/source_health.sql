-- name: RecordSourceAttempt :one
INSERT INTO source_health (slug, last_outcome, last_attempt_at, consecutive_failures)
VALUES (
    sqlc.arg(slug),
    sqlc.arg(last_outcome),
    sqlc.arg(last_attempt_at),
    CASE WHEN sqlc.arg(last_outcome)::text = 'error' THEN 1 ELSE 0 END
)
ON CONFLICT (slug) DO UPDATE
    SET last_outcome = EXCLUDED.last_outcome,
        last_attempt_at = EXCLUDED.last_attempt_at,
        -- The count runs in the database, so two lookups at once cannot both read the same total.
        consecutive_failures = CASE
            WHEN EXCLUDED.last_outcome = 'error' THEN source_health.consecutive_failures + 1
            ELSE 0
        END
RETURNING slug, last_outcome, last_attempt_at, consecutive_failures;

-- name: ListSourceHealth :many
SELECT slug, last_outcome, last_attempt_at, consecutive_failures
FROM source_health;
