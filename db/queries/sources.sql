-- name: UpsertSourceState :one
-- Record the operator's on/off choice for one source. A toggle is a Declared act
-- with no timeline, so re-toggling overwrites the single current value rather
-- than appending, and toggled_at re-stamps to when the current state was set.
INSERT INTO source_state (slug, enabled, toggled_by)
VALUES ($1, $2, $3)
ON CONFLICT (slug) DO UPDATE
    SET enabled = EXCLUDED.enabled,
        toggled_by = EXCLUDED.toggled_by,
        toggled_at = now()
RETURNING slug, enabled, toggled_by, toggled_at;

-- name: ListSourceStates :many
-- The operator's overrides of the authored ship defaults. The handler merges
-- these onto the in-binary catalogue: a source's effective state is its override
-- where one exists and its shipped default otherwise.
SELECT s.slug, s.enabled, s.toggled_by, s.toggled_at,
       a.username AS toggled_by_username
FROM source_state s
JOIN account a ON a.id = s.toggled_by;
