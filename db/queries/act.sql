-- name: InsertAct :exec
-- id and created_at are defaulted: the caller cannot forge the time (spec §4.1).
INSERT INTO act (actor_kind, actor, action, subject)
VALUES ($1, $2, $3, $4);

-- name: ListActsInRange :many
-- A client-side scope reaches only the rows sent, and the corpus is unbounded (ADR-0158 limb 4).
SELECT id, created_at, actor_kind, actor, action, subject
FROM act
WHERE created_at >= sqlc.arg(from_time) AND created_at < sqlc.arg(until_time)
ORDER BY created_at DESC, id DESC;
