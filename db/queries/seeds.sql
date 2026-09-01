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
-- Withdraws a declared Seed by id (#21a: the Scope chip-remove act), and records
-- the tombstone an ADDRESS withdrawal owes (ADR-0134 §2, #1040). A viewer never
-- reaches it (requireAdmin). Idempotent: withdrawing a row already gone deletes
-- nothing, writes no tombstone and is not an error, so a stale chip submit is a
-- no-op.
--
-- The delete and the tombstone are ONE statement, so they commit together and no
-- path can leave a withdrawn scope with no mover for the membership fold to name.
-- A data-modifying CTE runs exactly once and always to completion, whether or not
-- the primary query reads it, so `tombstone` fires on its own.
--
-- The tombstone is written for an `address` Seed alone. A name Seed's withdrawal
-- is a different message contract and stays the gap ADR-0134 §7 names, so it
-- deletes exactly as it did before.
WITH removed AS (
    DELETE FROM seed WHERE seed.id = sqlc.arg(seed_id) RETURNING seed.kind, seed.address_cidr
),
tombstone AS (
    INSERT INTO seed_withdrawal (address_cidr, created_by)
    SELECT r.address_cidr, sqlc.arg(created_by)
    FROM removed r
    WHERE r.kind = 'address' AND r.address_cidr IS NOT NULL
    RETURNING 1 AS written
)
SELECT
    (SELECT count(*) FROM removed)::bigint   AS seeds_removed,
    (SELECT count(*) FROM tombstone)::bigint AS tombstones_written;
