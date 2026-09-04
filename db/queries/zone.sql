-- name: CreateZoneFile :one
INSERT INTO zone_file (seed_id, supplied_at, content, uploaded_by)
VALUES ($1, $2, $3, $4)
RETURNING id, supplied_at;

-- name: LatestZoneFilesForDispatch :many
SELECT DISTINCT ON (z.seed_id)
    z.seed_id, s.name_domain, z.supplied_at, z.content
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
WHERE s.kind = 'name'
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;

-- name: ListZoneFileStatus :many
SELECT DISTINCT ON (z.seed_id)
    z.seed_id, s.name_domain, z.supplied_at, z.created_at,
    a.username AS uploaded_by_username,
    length(z.content)::bigint AS content_bytes
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
JOIN account a ON a.id = z.uploaded_by
WHERE s.kind = 'name'
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;

-- name: GetZoneCadenceSeconds :one
SELECT cadence_seconds FROM scan WHERE kind = 'zone';

-- name: SetZoneCadenceSeconds :exec
-- A non-positive interval is refused by the table's CHECK, not by this statement.
UPDATE scan SET cadence_seconds = $1 WHERE kind = 'zone';
