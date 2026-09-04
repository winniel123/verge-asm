-- name: CreateSession :one
INSERT INTO session (account_id, token_hash, user_agent, ip, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, account_id, token_hash, created_at, last_seen_at, user_agent, ip, expires_at, revoked_at;

-- name: GetSessionByTokenHash :one
SELECT id, account_id, token_hash, created_at, last_seen_at, user_agent, ip, expires_at, revoked_at
FROM session
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2;

-- name: TouchSession :exec
UPDATE session SET last_seen_at = $2 WHERE id = $1;

-- name: RevokeSession :exec
UPDATE session SET revoked_at = $3
WHERE id = $1 AND account_id = $2 AND revoked_at IS NULL;

-- name: RevokeSessionByIDForAdmin :exec
UPDATE session SET revoked_at = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeOtherSessionsForAccount :exec
UPDATE session SET revoked_at = $3
WHERE account_id = $1 AND id <> $2 AND revoked_at IS NULL;

-- name: RevokeAllSessionsForAccount :exec
UPDATE session SET revoked_at = $2
WHERE account_id = $1 AND revoked_at IS NULL;

-- name: ListSessionsForAccount :many
SELECT id, account_id, created_at, last_seen_at, user_agent, ip, expires_at
FROM session
WHERE account_id = $1 AND revoked_at IS NULL AND expires_at > $2
ORDER BY last_seen_at DESC, id DESC;

-- name: ListAllActiveSessions :many
SELECT s.id, s.account_id, a.username, a.role, s.created_at, s.last_seen_at,
       s.user_agent, s.ip, s.expires_at
FROM session s
JOIN account a ON a.id = s.account_id
WHERE s.revoked_at IS NULL AND s.expires_at > $1
ORDER BY a.username ASC, s.last_seen_at DESC, s.id DESC;
