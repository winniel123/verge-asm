-- +goose Up
-- A recovery code is a single-use fallback second factor, minted as a set when an
-- account enrols in two-factor (SignIn's TOTP enrollment, #314): if the
-- authenticator is lost, one code stands in for the rotating TOTP at the login
-- two-factor step. It is account state, keyed by account_id with ON DELETE CASCADE.
--
-- Verge keeps only a hash. The plaintext codes are shown once, on the enrollment
-- screen right after the app is confirmed, and never again — only their SHA-256
-- hashes are stored, so a leaked database yields no usable code, and a lost set is
-- re-issued (which clears the old set), never recovered. The reveal-once contract
-- is the whole point: no query ever returns the plaintext, because it is not kept.
--
-- used_at is NULL until a code is redeemed, then carries the instant it was, so each
-- code works exactly once. (account_id, code_hash) is unique so one account cannot
-- hold the same code twice.
CREATE TABLE recovery_code (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    code_hash  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at    TIMESTAMPTZ,
    UNIQUE (account_id, code_hash)
);

-- +goose Down
DROP TABLE recovery_code;
