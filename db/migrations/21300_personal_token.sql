-- +goose Up
-- A personal API token is a per-account credential the operator mints for their
-- own automation (v1 spec §4.3): it is scoped to one account and inherits that
-- account's role, so it is account state and belongs beside the account, keyed by
-- account_id with ON DELETE CASCADE so deleting an account takes its tokens with it.
--
-- Verge keeps only a hash. The plaintext token is shown once, at mint, and never
-- again — only its bcrypt-class hash is stored, so a leaked database yields no
-- usable token, and a lost token is re-minted, never recovered. The prefix is the
-- short, non-secret head of the token (e.g. vg_pat_9f3k…) kept in the clear purely
-- so the operator can tell one row from another in the list; it is not sufficient
-- to authenticate.
--
-- last_used_at is nullable and starts NULL: a freshly minted token has genuinely
-- never been presented, which renders as an em dash rather than a fabricated
-- recency. (name, account_id) is unique so one account cannot mint two tokens under
-- the same label, which is what lets a typed-name revoke gate identify the target.
CREATE TABLE personal_token (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id   BIGINT NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL,
    token_hash   TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    UNIQUE (account_id, name)
);

-- +goose Down
DROP TABLE personal_token;
