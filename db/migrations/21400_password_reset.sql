-- +goose Up
-- A password reset is a single-use, time-boxed grant to set a new password without
-- the current one (SignIn's forgot/reset flow, #314). It is account state and lives
-- beside the account, keyed by account_id with ON DELETE CASCADE so deleting an
-- account takes its outstanding resets with it.
--
-- Verge keeps only a hash of the reset token. The plaintext rides one URL, handed
-- to the operator out of band (on a self-hosted host with no mail configured it is
-- written to the web logs, exactly as the first-boot setup token is), and only its
-- SHA-256 hash is stored — a leaked database yields no usable link. token_hash is
-- unique so one minted token names at most one row.
--
-- expires_at bounds the window; consumed_at is NULL until the reset is spent, then
-- carries the instant it was, so a spent or expired link is refused rather than
-- silently reusable. The single-use guarantee is the consumed_at stamp, checked
-- against the server clock the handler threads in.
CREATE TABLE password_reset (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id  BIGINT NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ
);

-- +goose Down
DROP TABLE password_reset;
