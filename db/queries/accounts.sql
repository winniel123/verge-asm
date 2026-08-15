-- name: CountAccounts :one
SELECT count(*) FROM account;

-- name: CreateAccount :one
INSERT INTO account (username, role, password_hash)
VALUES ($1, $2, $3)
RETURNING id, username, role, password_hash, totp_secret, totp_enabled, created_at;

-- name: GetAccountByUsername :one
SELECT id, username, role, password_hash, totp_secret, totp_enabled, created_at
FROM account
WHERE username = $1;

-- name: GetAccountByID :one
SELECT id, username, role, password_hash, totp_secret, totp_enabled, created_at
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

-- name: SetTOTPSecret :exec
UPDATE account SET totp_secret = $2, totp_enabled = false WHERE id = $1;

-- name: ConfirmTOTP :exec
UPDATE account SET totp_enabled = true WHERE id = $1 AND totp_secret IS NOT NULL;
