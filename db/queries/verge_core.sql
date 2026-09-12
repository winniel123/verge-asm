-- name: UpsertVergeCoreFrequencyEdit :exec
INSERT INTO verge_core_frequency_edit (port, action, created_by)
VALUES ($1, $2, $3)
ON CONFLICT (port) DO UPDATE SET action = EXCLUDED.action, created_by = EXCLUDED.created_by, created_at = now();

-- name: DeleteVergeCoreFrequencyEdit :one
-- The port rides the reset's own RETURNING, so a port carrying no edit returns nothing.
DELETE FROM verge_core_frequency_edit WHERE port = $1
RETURNING port;

-- name: ListVergeCoreFrequencyEditsWithAuthor :many
SELECT e.id, e.port, e.action, e.created_at
FROM verge_core_frequency_edit e
ORDER BY e.port;
