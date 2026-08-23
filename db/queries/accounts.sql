-- name: CountAccounts :one
SELECT count(*) FROM account;

-- name: CreateAccount :one
INSERT INTO account (username, role, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, role, password_hash, totp_secret, totp_enabled, created_at, totp_last_step;

-- name: GetAccountByUsername :one
SELECT id, username, role, password_hash, totp_secret, totp_enabled, created_at, totp_last_step
FROM account
WHERE username = $1;

-- name: GetAccountByID :one
SELECT id, username, role, password_hash, totp_secret, totp_enabled, created_at, totp_last_step
FROM account
WHERE id = $1;

-- name: ListAccounts :many
-- The accounts management list on the Settings screen. It omits password_hash
-- and totp_secret: managing accounts never needs either, so they stay out of the
-- render path.
SELECT id, username, role, totp_enabled, created_at
FROM account
ORDER BY created_at ASC, id ASC;

-- name: CountAdmins :one
-- Guards the last-admin invariant: a role change that would drop this to zero is
-- refused so an operator cannot lock every admin out.
SELECT count(*) FROM account WHERE role = 'admin';

-- name: UpdateAccountRole :exec
UPDATE account SET role = $2 WHERE id = $1;

-- name: UpdatePassword :exec
-- Change one account's own password (Profile → Credentials). The handler verifies
-- the current password and the new-password rules before this runs, so this is the
-- bare write; it never touches the TOTP secret, so a password change leaves the
-- second factor in force.
UPDATE account SET password_hash = $2 WHERE id = $1;

-- name: SetTOTPSecret :exec
UPDATE account SET totp_secret = $2, totp_enabled = false WHERE id = $1;

-- name: ConfirmTOTP :exec
UPDATE account SET totp_enabled = true WHERE id = $1 AND totp_secret IS NOT NULL;

-- name: SetTOTPLastStep :execrows
-- Atomically spend the TOTP step just accepted at login (#323, #339). The predicate
-- makes the advance the single serialisation point: the write lands only when the
-- account's stored watermark is still NULL or strictly below the presented step, so
-- of two concurrent requests carrying the SAME valid code exactly one updates a row
-- and the other affects zero — the loser is refused as a replay. A read-then-write in
-- the handler could let both pass; this conditional UPDATE cannot (RFC 6238 §5.2).
UPDATE account SET totp_last_step = $2
WHERE id = $1 AND (totp_last_step IS NULL OR totp_last_step < $2);

-- name: DeleteAccount :exec
-- Remove a member (Settings -> Team, T18). The handler gates this behind a typed-
-- name confirmation and refuses to remove yourself or the last admin. Attributed
-- work keeps the account's id: the created_by references on seeds, channels,
-- exclusions and the rest are NOT NULL with no cascade, so this deletes only an
-- account that authored none of them — the FK violation surfaces as a clear refusal
-- rather than a silent orphaning. The single-use pre-auth grants (personal tokens,
-- password resets, recovery codes) cascade; an invite the account issued or accepted
-- keeps its record with the reference nulled (ON DELETE SET NULL).
DELETE FROM account WHERE id = $1;

-- name: ResetAccountTOTP :exec
-- Require re-enrollment (Settings -> Team, T18): clear an account's second factor so
-- their current authenticator stops working at once and the next sign-in walks them
-- through TOTP setup again. It touches neither the password nor any session — a
-- signed-in account stays signed in until its cookie lapses. Symmetric to
-- SetTOTPSecret, which arms a fresh secret; this disarms the factor entirely.
UPDATE account SET totp_secret = NULL, totp_enabled = false WHERE id = $1;
