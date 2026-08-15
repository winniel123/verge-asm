-- name: SetCustodyExtension :exec
-- Declare (true) or withdraw (false) the custody extension on a name-scope Seed.
-- The kind guard makes the act a no-op on an address scope rather than an error,
-- matching the CHECK the migration installs — an address scope can never carry a
-- custody extension. The flag has no timeline, so a withdrawal is the same UPDATE
-- with false rather than a dated state change.
UPDATE seed
SET custody_extension = $2
WHERE id = $1 AND kind = 'name';
