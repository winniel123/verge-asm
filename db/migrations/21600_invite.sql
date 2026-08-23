-- +goose Up
-- An invite is a single-use, time-boxed grant to create one account at a given
-- role by presenting a token, no existing credentials required (SignIn's invite
-- acceptance, #314). This migration ships the entity and the acceptance side; the
-- creation side (Settings -> Team) lands in T18 and mints rows against this table.
--
-- Accounts on this build are usernames, not emails, and there is no identity
-- provider, so an invite carries no address to send to and no pre-bound username:
-- it carries the role the new account will hold and who issued it. The acceptor
-- chooses their own username and password on the set-credentials screen. Verge
-- keeps only a hash of the token; the plaintext rides one URL handed out of band.
--
-- expires_at bounds the window; consumed_at is NULL until the invite is accepted,
-- then carries the instant, and accepted_account_id records which account it
-- created — so a spent or expired invite is refused, never reusable. invited_by and
-- accepted_account_id are ON DELETE SET NULL so an invite outlives either account's
-- deletion as a record rather than cascading away.
CREATE TABLE invite (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_hash          TEXT NOT NULL UNIQUE,
    role                TEXT NOT NULL,
    invited_by          BIGINT REFERENCES account(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    consumed_at         TIMESTAMPTZ,
    accepted_account_id BIGINT REFERENCES account(id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE invite;
