-- name: GetInstanceConfig :one
-- The single operator-global row seeded by the migration; it always exists. Both
-- feature clusters (#390 API surfaces, #391 backup & updates) read their flags and
-- cached facts through this one row.
SELECT api_enabled, api_updated_by, api_updated_at,
       update_check_enabled, update_check_updated_by, update_check_updated_at,
       release_state, release_latest_version, release_latest_notes, release_checked_at,
       last_backup_at, last_backup_size,
       seed_address_cap, seed_address_cap_updated_by, seed_address_cap_updated_at
FROM instance_config
WHERE id = true;

-- name: SetAPIEnabled :exec
-- Flip the read-only /api/v1 surface on or off, stamping who acted and when so the
-- settings card can render the dated act of the current state (#390). Off by default;
-- a minted token stays inert until this is true.
UPDATE instance_config
SET api_enabled = $1, api_updated_by = $2, api_updated_at = now()
WHERE id = true;

-- name: SetUpdateCheckEnabled :exec
-- Opt the worker's daily release-feed check in or out, stamping who acted and when
-- (#391). While false the worker never dispatches the check — air-gap-safe.
UPDATE instance_config
SET update_check_enabled = $1, update_check_updated_by = $2, update_check_updated_at = now()
WHERE id = true;

-- name: SetReleaseCache :exec
-- Record the last result of the worker's best-effort release check (#391): the state
-- (current | newer | disabled) and, for a "newer", the latest version and notes. The
-- check instant is stamped now(), so a "checked N ago" reads honestly.
UPDATE instance_config
SET release_state = $1, release_latest_version = $2, release_latest_notes = $3,
    release_checked_at = now()
WHERE id = true;

-- name: SetLastBackup :exec
-- Record the last backup taken from the UI (#391): its instant (now()) and byte size,
-- surfaced on the Backup card.
UPDATE instance_config
SET last_backup_at = now(), last_backup_size = $1
WHERE id = true;

-- name: SetSeedAddressCap :exec
-- Set the operator address-scope cap (#888 / Settings #206, ADR-0127), stamping who
-- acted and when so the Settings control renders the dated act of the current cap.
-- The value is read at declaration only (ADR-0047 §5.3), so lowering it never
-- invalidates a scope declared under a higher cap. ADR-0127: no upper bound is
-- enforced here — the handler floors it at 1 and the column has no ceiling.
UPDATE instance_config
SET seed_address_cap = $1, seed_address_cap_updated_by = $2, seed_address_cap_updated_at = now()
WHERE id = true;
