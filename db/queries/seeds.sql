-- name: CreateNameSeed :one
INSERT INTO seed (kind, name_domain, created_by)
VALUES ('name', $1, $2)
RETURNING id, kind, name_domain, address_cidr, created_by, created_at, custody_extension;

-- name: CreateAddressSeed :one
INSERT INTO seed (kind, address_cidr, created_by)
VALUES ('address', $1, $2)
RETURNING id, kind, name_domain, address_cidr, created_by, created_at, custody_extension;

-- name: ListSeeds :many
SELECT s.id, s.kind, s.name_domain, s.address_cidr, s.custody_extension,
       s.created_by, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
ORDER BY s.created_at DESC, s.id DESC;

-- name: WithdrawSeed :one
-- A data-modifying CTE fires on its own, so dropping its count from the SELECT keeps the write.
WITH removed AS (
    DELETE FROM seed WHERE seed.id = sqlc.arg(seed_id)
    RETURNING seed.kind, seed.address_cidr, seed.name_domain
),
tombstone AS (
    INSERT INTO seed_withdrawal (kind, address_cidr, name_domain, created_by)
    SELECT r.kind, r.address_cidr, r.name_domain, sqlc.arg(created_by)
    FROM removed r
    WHERE (r.kind = 'address' AND r.address_cidr IS NOT NULL)
       OR (r.kind = 'name' AND r.name_domain IS NOT NULL)
    RETURNING 1 AS written
)
SELECT
    (SELECT count(*) FROM removed)::bigint   AS seeds_removed,
    (SELECT count(*) FROM tombstone)::bigint AS tombstones_written;
