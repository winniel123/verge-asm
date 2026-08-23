-- name: CreateSession :one
-- Open a session at login. Only the token's hash is stored; the opaque plaintext
-- lives solely in the cookie on the client (ADR-0117). Returns the row so the
-- caller holds the id it just minted.
INSERT INTO session (account_id, token_hash, user_agent, ip, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, account_id, token_hash, created_at, last_seen_at, user_agent, ip, expires_at, revoked_at;

-- name: GetSessionByTokenHash :one
-- The per-request validation lookup: resolve a presented session token (by its
-- hash) to a live row. A session is live only when it is unrevoked and unexpired,
-- so both gates are in SQL and a dead session simply returns no row — the handler
-- then treats it exactly as an absent cookie. The clock bound is passed in ($2) so
-- a fixed-clock test and production agree on the boundary.
SELECT id, account_id, token_hash, created_at, last_seen_at, user_agent, ip, expires_at, revoked_at
FROM session
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2;

-- name: TouchSession :exec
-- Refresh last_seen_at for the "last active" column. Called at most once per minute
-- per session (the handler throttles) so a busy session does not amplify writes.
UPDATE session SET last_seen_at = $2 WHERE id = $1;

-- name: RevokeSession :exec
-- Revoke one session, scoped to its owner: the account_id predicate means an
-- account can only revoke its own sessions, never another's by guessing an id —
-- the same owner-scoping personal-token revocation uses. Idempotent: a
-- revoked/absent row is unaffected.
UPDATE session SET revoked_at = $3
WHERE id = $1 AND account_id = $2 AND revoked_at IS NULL;

-- name: RevokeSessionByIDForAdmin :exec
-- Admin revocation of any single session by id, not owner-scoped — gated by
-- requireAdmin at the handler, never reachable by a viewer. Idempotent.
UPDATE session SET revoked_at = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeOtherSessionsForAccount :exec
-- "Sign out other devices" and password-change invalidation: revoke every live
-- session for the account EXCEPT the one making the request ($2, the current
-- session id). The current session survives so the acting user is not signed out
-- of the tab they are working in.
UPDATE session SET revoked_at = $3
WHERE account_id = $1 AND id <> $2 AND revoked_at IS NULL;

-- name: RevokeAllSessionsForAccount :exec
-- Revoke every live session for an account with no exception — the password-reset
-- path (no current session to keep) and the admin offboarding action. Idempotent.
UPDATE session SET revoked_at = $2
WHERE account_id = $1 AND revoked_at IS NULL;

-- name: ListSessionsForAccount :many
-- One account's live sessions, newest activity first — the Profile's personal
-- sessions list. token_hash is omitted from the read: listing never needs it, so
-- the secret material stays out of the render path.
SELECT id, account_id, created_at, last_seen_at, user_agent, ip, expires_at
FROM session
WHERE account_id = $1 AND revoked_at IS NULL AND expires_at > $2
ORDER BY last_seen_at DESC, id DESC;

-- name: ListAllActiveSessions :many
-- Every account's live sessions for the admin surface, joined to the account so the
-- view can show whose session it is and at what role. Ordered by account then
-- recency. token_hash is never selected here either.
SELECT s.id, s.account_id, a.username, a.role, s.created_at, s.last_seen_at,
       s.user_agent, s.ip, s.expires_at
FROM session s
JOIN account a ON a.id = s.account_id
WHERE s.revoked_at IS NULL AND s.expires_at > $1
ORDER BY a.username ASC, s.last_seen_at DESC, s.id DESC;
