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

-- name: DeleteSeed :execrows
-- Withdraws a declared Seed by id (#21a: the Scope chip-remove act). A viewer
-- never reaches it (requireAdmin). Idempotent: deleting a row already gone
-- affects zero rows and is not an error, so a stale chip submit is a no-op.
DELETE FROM seed WHERE id = $1;
