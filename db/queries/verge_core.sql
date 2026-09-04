-- name: UpsertVergeCoreFrequencyEdit :exec
INSERT INTO verge_core_frequency_edit (port, action, created_by)
VALUES ($1, $2, $3)
ON CONFLICT (port) DO UPDATE SET action = EXCLUDED.action, created_by = EXCLUDED.created_by, created_at = now();

-- name: DeleteVergeCoreFrequencyEdit :exec
DELETE FROM verge_core_frequency_edit WHERE port = $1;

-- name: ListVergeCoreFrequencyEditsWithAuthor :many
SELECT e.id, e.port, e.action, e.created_at, a.username AS created_by_username
FROM verge_core_frequency_edit e
JOIN account a ON a.id = e.created_by
ORDER BY e.port;
