-- name: UpsertSourceState :one
-- Record the operator's on/off choice for one source. A toggle is a Declared act
-- with no timeline, no actor, and no instant of its own (ADR-0073, ADR-0093), so
-- re-toggling overwrites the single current value and the row holds only the
-- overridden state.
INSERT INTO source_state (slug, enabled)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE
    SET enabled = EXCLUDED.enabled
RETURNING slug, enabled;

-- name: ListSourceStates :many
-- The operator's overrides of the authored ship defaults. The handler merges
-- these onto the in-binary catalogue: a source's effective state is its override
-- where one exists and its shipped default otherwise.
SELECT slug, enabled
FROM source_state;
