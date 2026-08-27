-- name: UpsertIntegrationState :one
-- Record the operator's install choice for one integration. An install is a
-- Declared act with no timeline, no actor, and no instant of its own (ADR-0073,
-- ADR-0093), so re-installing overwrites the single current state and the row
-- holds only the current install state. The channel binding is NOT touched here:
-- a re-install keeps whatever delivery Channel the integration was bound to (the
-- ON CONFLICT omits channel_id, leaving the existing value in place), and a first
-- install lands unbound (channel_id defaults NULL).
INSERT INTO integration_state (slug, state)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET state = EXCLUDED.state
RETURNING slug, state, channel_id;

-- name: ListIntegrationStates :many
-- The operator's install states and their bound delivery Channel, merged by the
-- handler onto the in-binary integration catalogue: an integration's effective state
-- is its stored state where a row exists and available (not installed) otherwise, and
-- its bound Channel (nullable — NULL is unbound) fills the drawer's BoundChannel hole.
SELECT slug, state, channel_id
FROM integration_state;

-- name: GetIntegrationChannel :one
-- The delivery Channel one integration is bound to (nullable — NULL is unbound). The
-- Send-test handler reads this to resolve where the test payload goes; an unbound
-- integration has nothing to send through.
SELECT channel_id
FROM integration_state
WHERE slug = $1;

-- name: SetIntegrationChannel :exec
-- Bind an installed integration to a delivery Channel, or clear the binding (a NULL
-- channel_id unbinds). Only an installed integration has a row to update; binding an
-- integration with no row is a no-op (an available integration cannot bind, and the
-- drawer offers it no channel select).
UPDATE integration_state
SET channel_id = $2
WHERE slug = $1;

-- name: DeleteIntegrationState :exec
-- Disconnect an integration, returning it to available (not installed). Absence of
-- a row is the available state, so a disconnect removes the row rather than storing
-- a sentinel.
DELETE FROM integration_state
WHERE slug = $1;
