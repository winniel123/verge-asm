-- name: CreateInvite :one
INSERT INTO invite (token_hash, role, invited_by, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, token_hash, role, invited_by, created_at, expires_at, consumed_at, accepted_account_id;

-- name: GetInviteByTokenHash :one
SELECT id, token_hash, role, invited_by, created_at, expires_at, consumed_at, accepted_account_id
FROM invite
WHERE token_hash = $1;

-- name: ConsumeInvite :execrows
-- The guard makes the consume atomic, so two accepts in flight cannot both win (#1650).
UPDATE invite SET consumed_at = $2, accepted_account_id = $3
WHERE id = $1 AND consumed_at IS NULL;
