-- name: CreateNameExclusion :one
-- kind is 'name' (an exact FQDN) or 'subtree' (that name and everything beneath).
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

-- name: DeleteExclusion :exec
-- Un-excluding removes the row: an exclusion is Declared input with no timeline,
-- so withdrawing it is a delete rather than a state change.
DELETE FROM exclusion WHERE id = $1;
