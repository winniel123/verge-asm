-- name: CreatePasswordReset :one
INSERT INTO password_reset (account_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, account_id, token_hash, created_at, expires_at, consumed_at;

-- name: GetPasswordResetByHash :one
SELECT id, account_id, token_hash, created_at, expires_at, consumed_at
FROM password_reset
WHERE token_hash = $1;

-- name: ConsumePasswordReset :exec
UPDATE password_reset SET consumed_at = $2 WHERE id = $1;
