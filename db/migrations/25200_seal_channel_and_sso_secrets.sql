-- +goose Up
-- Both columns now hold AEAD ciphertext, and every reader refuses a value it cannot open
-- (ADR-0172 §2, #1679). A pre-seal cleartext row can never open, so it is cleared here,
-- and the operator re-enters it on Settings as a restore already asks (ADR-0160).
UPDATE channel SET secret = NULL, updated_at = now() WHERE secret IS NOT NULL;
UPDATE sso_provider SET client_secret = NULL, updated_at = now() WHERE client_secret IS NOT NULL;

-- +goose Down
-- The cleared cleartext is unrecoverable by design, so the columns stay NULL.
SELECT 1;
