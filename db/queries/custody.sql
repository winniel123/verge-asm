-- name: SetCustodyExtension :one
UPDATE seed
SET custody_extension = $2
-- The seed CHECK rejects a true extension on an address scope, so an unguarded declare errors.
WHERE id = $1 AND kind = 'name'
-- The scope rides the move's own RETURNING, so an address id moves nothing and returns nothing.
RETURNING name_domain;
