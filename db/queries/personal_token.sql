-- name: CreatePersonalToken :one
INSERT INTO personal_token (account_id, name, prefix, token_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, account_id, name, prefix, token_hash, created_at, last_used_at;

-- name: ListPersonalTokens :many
SELECT id, account_id, name, prefix, created_at, last_used_at
FROM personal_token
WHERE account_id = $1
ORDER BY created_at DESC, id DESC;

-- name: DeletePersonalToken :exec
DELETE FROM personal_token
WHERE id = $1 AND account_id = $2;

-- name: GetPersonalTokenByHash :one
SELECT id, account_id, name, prefix, token_hash, created_at, last_used_at
FROM personal_token
WHERE token_hash = $1;

-- name: UpdatePersonalTokenLastUsed :exec
UPDATE personal_token
SET last_used_at = now()
WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < now() - interval '1 hour');
