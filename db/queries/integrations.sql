-- name: UpsertIntegrationState :one
INSERT INTO integration_state (slug, state)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET state = EXCLUDED.state
RETURNING slug, state, channel_id;

-- name: ListIntegrationStates :many
SELECT slug, state, channel_id
FROM integration_state;

-- name: GetIntegrationChannel :one
SELECT channel_id
FROM integration_state
WHERE slug = $1;

-- name: SetIntegrationChannel :exec
UPDATE integration_state
SET channel_id = $2
WHERE slug = $1;

-- name: DeleteIntegrationState :exec
DELETE FROM integration_state
WHERE slug = $1;
