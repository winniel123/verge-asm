-- name: CreateRecoveryCode :exec
INSERT INTO recovery_code (account_id, code_hash) VALUES ($1, $2);

-- name: DeleteRecoveryCodesForAccount :exec
DELETE FROM recovery_code WHERE account_id = $1;

-- name: ListUnusedRecoveryCodeHashes :many
SELECT id, code_hash FROM recovery_code
WHERE account_id = $1 AND used_at IS NULL
ORDER BY id ASC;

-- name: ConsumeRecoveryCode :exec
UPDATE recovery_code SET used_at = $2 WHERE id = $1;
