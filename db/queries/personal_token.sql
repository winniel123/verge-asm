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

-- name: GetPersonalTokenByHash :one
-- Resolve a presented bearer credential to its stored row by the SHA-256 hash of the
-- plaintext vg_pat_… (the caller hashes before this lookup; the plaintext is never
-- persisted, only its digest is). The indexed hash equality carries the constant-time
-- property inherently — a non-matching hash simply yields no row, disclosing nothing by
-- timing. Returns account_id so the bearer path reads the account's role LIVE per request
-- (ADR-0123 §4), never freezing a role into the token itself.
SELECT id, account_id, name, prefix, token_hash, created_at, last_used_at
FROM personal_token
WHERE token_hash = $1;

-- name: UpdatePersonalTokenLastUsed :exec
-- Coarsened last-used touch (ADR-0123 §4): stamp last_used_at = now() at most once per
-- hour per token, so an authenticated /api/v1 request records "still live" without a
-- row-per-request write amplifier and without turning last_used_at into a fine-grained
-- access log of the operator's own integration traffic. The predicate makes the write a
-- no-op inside the hour, and last_used_at never regresses — it is data and rides the backup.
UPDATE personal_token
SET last_used_at = now()
WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < now() - interval '1 hour');
