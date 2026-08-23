-- name: CreateInvite :one
-- Mint a single-use, time-boxed invite at a role (Settings -> Team, T18). This is
-- the creation side of the invite table T19 shipped for acceptance: web keeps only
-- a hash of the token, and the plaintext rides one join URL handed out of band.
-- invited_by attributes the issuing admin so the invite outlives them as a record
-- (ON DELETE SET NULL); expires_at bounds the window. The row starts unconsumed —
-- consumed_at and accepted_account_id stay NULL until the acceptance screen spends
-- it (ConsumeInvite).
INSERT INTO invite (token_hash, role, invited_by, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, token_hash, role, invited_by, created_at, expires_at, consumed_at, accepted_account_id;

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
