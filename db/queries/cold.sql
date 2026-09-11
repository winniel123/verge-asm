-- name: ListColdScopeSeeds :many
SELECT s.id, s.kind, s.name_domain, s.address_cidr
FROM cold_scan_scope c
JOIN seed s ON s.id = c.seed_id
ORDER BY s.id;

-- name: ListColdScopeSeedIds :many
SELECT seed_id FROM cold_scan_scope ORDER BY seed_id;

-- name: OptInColdScope :one
-- A data-modifying CTE fires on its own, so the SELECT need only read the scope back.
WITH enrolled AS (
    INSERT INTO cold_scan_scope (seed_id, created_by)
    VALUES ($1, $2)
    ON CONFLICT (seed_id) DO NOTHING
    RETURNING seed_id
)
-- The scope rides the enrolment's own RETURNING, so a repeat opt-in returns nothing.
SELECT s.address_cidr, s.name_domain
FROM enrolled e
JOIN seed s ON s.id = e.seed_id;

-- name: OptOutColdScope :one
WITH withdrawn AS (
    DELETE FROM cold_scan_scope WHERE seed_id = $1
    RETURNING seed_id
)
-- The scope rides the withdrawal's own RETURNING, so an unenrolled scope returns nothing.
SELECT s.address_cidr, s.name_domain
FROM withdrawn w
JOIN seed s ON s.id = w.seed_id;

-- name: SyncColdScanEnabled :exec
UPDATE scan
SET enabled = EXISTS (SELECT 1 FROM cold_scan_scope)
WHERE kind = 'cold';
