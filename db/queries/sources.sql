-- name: UpsertSourceState :one
INSERT INTO source_state (slug, enabled)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET enabled = EXCLUDED.enabled
RETURNING slug, enabled;

-- name: ListSourceStates :many
SELECT slug, enabled
FROM source_state;
