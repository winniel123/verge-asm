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

-- name: ListAddressExclusionCidrs :many
-- The declared `address` exclusion CIDRs, for the Custody derivation: an address
-- inside one is NOT covered by the address-scope limb, so it derives third-party
-- unless a custody extension also reaches it (ADR-0012 §125, ADR-0133 §1).
--
-- It is a separate query from ListExclusions on purpose. That one returns all three
-- kinds joined to `account` for the chip render, and this is a batch-time read that
-- wants the CIDRs alone. It is the address twin of measurement.sql's
-- ListAddressScopeCidrs and mirrors its shape read for read.
SELECT address_cidr
FROM exclusion
WHERE kind = 'address' AND address_cidr IS NOT NULL
ORDER BY id;

-- name: DeleteExclusion :exec
-- Un-excluding removes the row: an exclusion is Declared input with no timeline,
-- so withdrawing it is a delete rather than a state change.
DELETE FROM exclusion WHERE id = $1;
