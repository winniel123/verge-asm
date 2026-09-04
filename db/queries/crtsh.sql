-- name: ListNameSeeds :many
SELECT id, name_domain
FROM seed
WHERE kind = 'name' AND name_domain IS NOT NULL
ORDER BY id;

-- name: InsertAdmittedName :exec
INSERT INTO admitted_name (name, source, seed_id, batch_id)
VALUES ($1, $2, $3, $4);

-- name: ListAdmittedNames :many
-- An append-only source re-admits the same names on every poll.
SELECT DISTINCT name
FROM admitted_name
ORDER BY name;

-- name: CTLastBatchAdmitCount :one
SELECT COALESCE((
    SELECT count(*)
    FROM admitted_name an
    WHERE an.batch_id = (
        SELECT b.id FROM batch b
        WHERE b.kind = 'ct'
        ORDER BY b.created_at DESC, b.id DESC
        LIMIT 1
    )
), 0)::bigint AS names;

-- name: ReserveCTSlot :one
WITH reserved AS (
    UPDATE ct_throttle
    SET next_free_at = GREATEST(next_free_at, now())
        + make_interval(secs => sqlc.arg(interval_seconds)::double precision)
    WHERE source = sqlc.arg(source)
    RETURNING next_free_at
)
SELECT (next_free_at
    - make_interval(secs => sqlc.arg(interval_seconds)::double precision))::timestamptz AS slot_at
FROM reserved;

-- name: ListAdmittedNamesOutsideSeed :many
SELECT DISTINCT name
FROM admitted_name
WHERE seed_id <> sqlc.arg(seed_id)
ORDER BY name;
