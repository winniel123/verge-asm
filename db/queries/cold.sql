-- name: ListColdScopeSeeds :many
-- The cold Scan's opted-in scope, for dispatch: every `Seed` opted into the
-- full-range tier, with its kind and scope so the dispatcher can bound the sweep
-- to the addresses an address-scope enumerates or a name-scope's names resolve
-- to. An empty result is the shipped disabled state — no jobs (ADR-0044).
SELECT s.id, s.kind, s.name_domain, s.address_cidr
FROM cold_scan_scope c
JOIN seed s ON s.id = c.seed_id
ORDER BY s.id;

-- name: ListColdScopeSeedIds :many
-- The opted-in `Seed` ids, for the Seeds screen to mark which scopes have opted
-- into the cold tier.
SELECT seed_id FROM cold_scan_scope ORDER BY seed_id;

-- name: OptInColdScope :exec
-- Opts one `Seed` scope into the cold tier. Idempotent on seed_id: opting an
-- already-opted-in scope in again is a no-op, never a duplicate.
INSERT INTO cold_scan_scope (seed_id, created_by)
VALUES ($1, $2)
ON CONFLICT (seed_id) DO NOTHING;

-- name: OptOutColdScope :exec
-- Opts one `Seed` scope back out of the cold tier.
DELETE FROM cold_scan_scope WHERE seed_id = $1;

-- name: SyncColdScanEnabled :exec
-- Reconciles the cold Scan's enabled flag with its scope: enabled exactly while
-- at least one `Seed` scope is opted in. This is the whole of "enabling it is
-- per-Seed, not global" (ADR-0044) — the operator never toggles a global switch;
-- opting the first scope in enables the tier, opting the last out disables it.
-- Called after every opt-in and opt-out, never on a cadence tick, so the tier is
-- never enabled as a side effect of a measurement.
UPDATE scan
SET enabled = EXISTS (SELECT 1 FROM cold_scan_scope)
WHERE kind = 'cold';
