-- name: CreatePasswordReset :one
-- Mint a single-use password-reset grant for one account. Only the token hash is
-- stored; the plaintext rides one URL handed to the operator out of band. The
-- caller sets expires_at from the server clock, so the window is bounded by the
-- same injectable clock every other auth read uses.
INSERT INTO password_reset (account_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, account_id, token_hash, created_at, expires_at, consumed_at;

-- name: GetPasswordResetByHash :one
-- Resolve a presented reset token to its row by hash. Validity (unconsumed,
-- unexpired) is checked in the handler against the server clock rather than SQL
-- now(), so a fixed-clock test and production agree on the same boundary.
SELECT id, account_id, token_hash, created_at, expires_at, consumed_at
FROM password_reset
WHERE token_hash = $1;

-- name: ConsumePasswordReset :exec
-- Spend a reset grant: stamp consumed_at with the instant the caller passes, which
-- makes it single-use. A second present of the same token then reads a non-NULL
-- consumed_at and is refused.
UPDATE password_reset SET consumed_at = $2 WHERE id = $1;
