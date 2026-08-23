-- name: GetInviteByTokenHash :one
-- Resolve a presented invite token to its row by hash. Validity (unconsumed,
-- unexpired) is checked in the handler against the server clock rather than SQL
-- now(), matching every other auth read's use of the injectable clock.
SELECT id, token_hash, role, invited_by, created_at, expires_at, consumed_at, accepted_account_id
FROM invite
WHERE token_hash = $1;

-- name: ConsumeInvite :exec
-- Spend an invite: stamp consumed_at with the instant the caller passes and record
-- which account the acceptance created, which makes it single-use. A second present
-- of the same token then reads a non-NULL consumed_at and is refused.
UPDATE invite SET consumed_at = $2, accepted_account_id = $3 WHERE id = $1;
