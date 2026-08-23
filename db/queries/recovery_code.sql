-- name: CreateRecoveryCode :exec
-- Store one recovery code's hash for an account. The plaintext is shown once at the
-- call site and never persisted; only the hash is kept, so it cannot be shown again.
INSERT INTO recovery_code (account_id, code_hash) VALUES ($1, $2);

-- name: DeleteRecoveryCodesForAccount :exec
-- Clear an account's recovery codes before re-issuing a set, so enrolling (or
-- re-enrolling) two-factor replaces the old codes wholesale rather than
-- accumulating stale sets that would each still redeem.
DELETE FROM recovery_code WHERE account_id = $1;

-- name: ListUnusedRecoveryCodeHashes :many
-- The account's still-redeemable codes, by id and hash, for the login fallback: the
-- handler hashes the presented code and matches it here. Used codes are excluded so
-- each redeems exactly once.
SELECT id, code_hash FROM recovery_code
WHERE account_id = $1 AND used_at IS NULL
ORDER BY id ASC;

-- name: ConsumeRecoveryCode :exec
-- Spend one recovery code: stamp used_at with the instant the caller passes. A
-- second present of the same code then reads a non-NULL used_at and is no longer
-- listed as redeemable.
UPDATE recovery_code SET used_at = $2 WHERE id = $1;
