-- name: CreateNameSeed :one
INSERT INTO seed (kind, name_domain, created_by)
VALUES ('name', $1, $2)
RETURNING id, kind, name_domain, address_cidr, created_by, created_at;

-- name: CreateAddressSeed :one
INSERT INTO seed (kind, address_cidr, created_by)
VALUES ('address', $1, $2)
RETURNING id, kind, name_domain, address_cidr, created_by, created_at;

-- name: ListSeeds :many
SELECT s.id, s.kind, s.name_domain, s.address_cidr, s.created_by, s.created_at,
       a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
ORDER BY s.created_at DESC, s.id DESC;
