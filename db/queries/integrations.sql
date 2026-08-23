-- name: UpsertIntegrationState :one
-- Record the operator's install choice for one integration. An install is a
-- Declared act with no timeline, no actor, and no instant of its own (ADR-0073,
-- ADR-0093), so re-installing overwrites the single current state and the row
-- holds only the current install state.
INSERT INTO integration_state (slug, state)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET state = EXCLUDED.state
RETURNING slug, state;

-- name: ListIntegrationStates :many
-- The operator's install states, merged by the handler onto the in-binary
-- integration catalogue: an integration's effective state is its stored state
-- where a row exists and available (not installed) otherwise.
SELECT slug, state
FROM integration_state;

-- name: DeleteIntegrationState :exec
-- Disconnect an integration, returning it to available (not installed). Absence of
-- a row is the available state, so a disconnect removes the row rather than storing
-- a sentinel.
DELETE FROM integration_state
WHERE slug = $1;
