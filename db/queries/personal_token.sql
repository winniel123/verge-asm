-- name: CreatePersonalToken :one
-- Mint a personal API token for one account. Only the hash and the non-secret
-- prefix are stored; the plaintext is shown once at the call site and never
-- persisted. A duplicate (account_id, name) is a unique violation, surfaced to the
-- operator as a name-already-taken message rather than a second silent row.
INSERT INTO personal_token (account_id, name, prefix, token_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, account_id, name, prefix, token_hash, created_at, last_used_at;

-- name: ListPersonalTokens :many
-- One account's tokens, newest first. token_hash is omitted from the read: listing
-- tokens never needs it, so the secret material stays out of the render path — only
-- the label, the non-secret prefix, and the timestamps are surfaced.
SELECT id, account_id, name, prefix, created_at, last_used_at
FROM personal_token
WHERE account_id = $1
ORDER BY created_at DESC, id DESC;

-- name: DeletePersonalToken :exec
-- Revoke a token, scoped to its owner: the account_id predicate means an operator
-- can only revoke their own tokens, never another account's by guessing an id.
-- Revocation is a hard delete — a revoked token holds no history worth reading.
DELETE FROM personal_token
WHERE id = $1 AND account_id = $2;
