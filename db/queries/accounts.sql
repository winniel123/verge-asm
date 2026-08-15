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

-- name: SetTOTPSecret :exec
UPDATE account SET totp_secret = $2, totp_enabled = false WHERE id = $1;

-- name: ConfirmTOTP :exec
UPDATE account SET totp_enabled = true WHERE id = $1 AND totp_secret IS NOT NULL;
