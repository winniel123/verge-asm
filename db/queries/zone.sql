-- name: CreateZoneFile :one
-- Records one supply act: a name-scope Seed's zone file at the operator's supply
-- instant. Append-only — a re-export is a new row, never an update.
INSERT INTO zone_file (seed_id, supplied_at, content, uploaded_by)
VALUES ($1, $2, $3, $4)
RETURNING id, supplied_at;

-- name: LatestZoneFilesForDispatch :many
-- The zone Scan's scope: the latest supplied file per name-scope Seed, with its
-- domain and supply instant, for the worker to restate. DISTINCT ON keeps only
-- the most recent supply per Seed.
SELECT DISTINCT ON (z.seed_id)
    z.seed_id, s.name_domain, z.supplied_at, z.content
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
WHERE s.kind = 'name'
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;

-- name: ListZoneFileStatus :many
-- The Seeds-screen view: the latest supplied file per name-scope Seed, without
-- the content, so the operator sees which scopes hold a zone file, when it was
-- supplied and by whom.
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
-- The operator's declared re-supply interval, held as the zone Scan's cadence.
SELECT cadence_seconds FROM scan WHERE kind = 'zone';

-- name: SetZoneCadenceSeconds :exec
-- Moves the re-supply interval dial. cadence_seconds > 0 is enforced by the
-- table's CHECK, so a non-positive interval is rejected by the database.
UPDATE scan SET cadence_seconds = $1 WHERE kind = 'zone';
