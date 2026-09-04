-- name: SetCustodyExtension :exec
UPDATE seed
SET custody_extension = $2
-- The seed CHECK rejects a true extension on an address scope, so an unguarded declare errors.
WHERE id = $1 AND kind = 'name';
