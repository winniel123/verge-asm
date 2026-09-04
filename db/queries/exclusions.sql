-- name: CreateNameExclusion :one
INSERT INTO exclusion (kind, name, created_by)
VALUES ($1, $2, $3)
RETURNING id, kind, name, address_cidr, created_by, created_at;

-- name: CreateAddressExclusion :one
INSERT INTO exclusion (kind, address_cidr, created_by)
VALUES ('address', $1, $2)
RETURNING id, kind, name, address_cidr, created_by, created_at;

-- name: ListExclusions :many
SELECT e.id, e.kind, e.name, e.address_cidr, e.created_by, e.created_at,
       a.username AS created_by_username
FROM exclusion e
JOIN account a ON a.id = e.created_by
ORDER BY e.created_at DESC, e.id DESC;

-- name: ListAddressExclusionCidrs :many
SELECT address_cidr
FROM exclusion
WHERE kind = 'address' AND address_cidr IS NOT NULL
ORDER BY id;

-- name: DeleteExclusion :exec
DELETE FROM exclusion WHERE id = $1;
