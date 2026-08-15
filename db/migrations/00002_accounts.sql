-- +goose Up
-- Local accounts are the only identity source: no SSO/OIDC/forward-auth, so
-- every authenticated act has a row here to attribute it to (v1 spec §4.3).
-- The TOTP secret and session signing key are the two auth secrets; the key
-- lives in the web-only volume, never here, but the per-account TOTP secret
-- is account state and belongs with the account.
CREATE TABLE account (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    role          TEXT NOT NULL CHECK (role IN ('admin', 'viewer')),
    password_hash TEXT NOT NULL,
    totp_secret   TEXT,
    totp_enabled  BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE account;
