-- name: GetInstanceConfig :one
-- The migration seeds this row, so no-rows is not a reachable state.
SELECT api_enabled, api_updated_by, api_updated_at,
       update_check_enabled, update_check_updated_by, update_check_updated_at,
       release_state, release_latest_version, release_latest_notes, release_checked_at,
       last_backup_at, last_backup_size,
       seed_address_cap, seed_address_cap_updated_by, seed_address_cap_updated_at
FROM instance_config
WHERE id = true;

-- name: SetAPIEnabled :exec
UPDATE instance_config
SET api_enabled = $1, api_updated_by = $2, api_updated_at = now()
WHERE id = true;

-- name: SetUpdateCheckEnabled :exec
UPDATE instance_config
SET update_check_enabled = $1, update_check_updated_by = $2, update_check_updated_at = now()
WHERE id = true;

-- name: SetReleaseCache :exec
UPDATE instance_config
SET release_state = $1, release_latest_version = $2, release_latest_notes = $3,
    release_checked_at = now()
WHERE id = true;

-- name: SetLastBackup :exec
UPDATE instance_config
SET last_backup_at = now(), last_backup_size = $1
WHERE id = true;

-- name: SetSeedAddressCap :exec
-- The operator cap has no upper bound; a large scope is priced at policy time (ADR-0127).
UPDATE instance_config
SET seed_address_cap = $1, seed_address_cap_updated_by = $2, seed_address_cap_updated_at = now()
WHERE id = true;
