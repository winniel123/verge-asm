-- name: CreateAnnotation :one
INSERT INTO annotation (subject_key, signal_name, reason)
VALUES ($1, $2, $3)
RETURNING id, subject_key, signal_name, reason, declared_at;

-- name: ListAnnotations :many
-- A dial is not ranked: no staleness sort and no per-rule count (ADR-0073 §3, §4).
SELECT id, subject_key, signal_name, reason, declared_at
FROM annotation
ORDER BY signal_name, subject_key;

-- name: DeleteAnnotation :exec
DELETE FROM annotation WHERE id = $1;
