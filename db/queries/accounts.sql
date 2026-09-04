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
SELECT id, username, role, totp_enabled, created_at
FROM account
ORDER BY created_at ASC, id ASC;

-- name: CountAdmins :one
SELECT count(*) FROM account WHERE role = 'admin';

-- name: UpdateAccountRole :exec
UPDATE account SET role = $2 WHERE id = $1;

-- name: UpdatePassword :exec
UPDATE account SET password_hash = $2 WHERE id = $1;

-- name: SetTOTPSecret :exec
UPDATE account SET totp_secret = $2, totp_enabled = false WHERE id = $1;

-- name: ConfirmTOTP :exec
UPDATE account SET totp_enabled = true WHERE id = $1 AND totp_secret IS NOT NULL;

-- name: SetTOTPLastStep :execrows
UPDATE account SET totp_last_step = $2
WHERE id = $1 AND (totp_last_step IS NULL OR totp_last_step < $2);

-- name: DeleteAccount :exec
DELETE FROM account WHERE id = $1;

-- name: ResetAccountTOTP :exec
UPDATE account SET totp_secret = NULL, totp_enabled = false WHERE id = $1;
