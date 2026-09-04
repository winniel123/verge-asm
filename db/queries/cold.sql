-- name: ListColdScopeSeeds :many
SELECT s.id, s.kind, s.name_domain, s.address_cidr
FROM cold_scan_scope c
JOIN seed s ON s.id = c.seed_id
ORDER BY s.id;

-- name: ListColdScopeSeedIds :many
SELECT seed_id FROM cold_scan_scope ORDER BY seed_id;

-- name: OptInColdScope :exec
INSERT INTO cold_scan_scope (seed_id, created_by)
VALUES ($1, $2)
ON CONFLICT (seed_id) DO NOTHING;

-- name: OptOutColdScope :exec
DELETE FROM cold_scan_scope WHERE seed_id = $1;

-- name: SyncColdScanEnabled :exec
UPDATE scan
SET enabled = EXISTS (SELECT 1 FROM cold_scan_scope)
WHERE kind = 'cold';
