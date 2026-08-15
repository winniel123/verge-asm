-- verge-core frequency-half editing (v1 spec §3.5). Only the frequency half is
-- operator-editable; these queries manage the delta rows the hot fan-out applies
-- over the shipped default. The sensitive half has no table and no query — it is
-- authored by the release and is unreachable from here by construction.

-- name: UpsertVergeCoreFrequencyEdit :exec
-- Record an operator edit to a frequency port. One row per port: a later edit
-- replaces the earlier one, so toggling add→remove on a port is an update, not a
-- second row.
INSERT INTO verge_core_frequency_edit (port, action, created_by)
VALUES ($1, $2, $3)
ON CONFLICT (port) DO UPDATE SET action = EXCLUDED.action, created_by = EXCLUDED.created_by, created_at = now();

-- name: DeleteVergeCoreFrequencyEdit :exec
-- Reset a port to its shipped default by dropping its edit row. Idempotent: a
-- port with no edit is already at its default.
DELETE FROM verge_core_frequency_edit WHERE port = $1;

-- name: ListVergeCoreFrequencyEditsWithAuthor :many
-- The current frequency edits, with who made each, for the management UI.
SELECT e.id, e.port, e.action, e.created_at, a.username AS created_by_username
FROM verge_core_frequency_edit e
JOIN account a ON a.id = e.created_by
ORDER BY e.port;
