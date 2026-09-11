-- name: InsertAct :exec
-- id and created_at are defaulted: the caller cannot forge the time (spec §4.1).
INSERT INTO act (actor_kind, actor, action, subject)
VALUES ($1, $2, $3, $4);

-- name: ListActsInRange :many
-- A client-side scope reaches only the rows sent, and the corpus is unbounded (ADR-0158 limb 4).
SELECT id, created_at, actor_kind, actor, action, subject
FROM act
WHERE created_at >= @from_time
  AND (sqlc.narg('until_time')::timestamptz IS NULL OR created_at < sqlc.narg('until_time')::timestamptz)
ORDER BY created_at DESC, id DESC;

-- name: AnyActRecorded :one
-- The period's empty state and the corpus's are different facts, and E.3 claims the second.
SELECT EXISTS (SELECT 1 FROM act);
