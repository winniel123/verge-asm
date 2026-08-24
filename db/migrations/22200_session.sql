-- +goose Up
-- A session is a server-side record so it can be revoked (ADR-0117). The cookie
-- carries an opaque high-entropy token, still HMAC-signed for tamper-evidence;
-- this row holds only that token's SHA-256 hash plus metadata, so a read-only
-- database leak discloses who was signed in and when, never a usable credential
-- (ADR-0053 preserved — the signing key still lives only on the web-state volume).
--
-- currentAccount validates a request's session against this row on every request:
-- it must exist, be unrevoked (revoked_at IS NULL), and be unexpired. Revoking a
-- session is setting revoked_at, and it takes effect on that session's next request.
--
-- account_id CASCADEs so deleting an account takes its sessions with it. token_hash
-- is UNIQUE and indexed (the per-request lookup); account_id is indexed (the
-- personal and admin listings). user_agent and ip are captured at issue for the
-- "which device / where from" columns the Profile and admin views render.
CREATE TABLE session (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id   BIGINT NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent   TEXT NOT NULL DEFAULT '',
    ip           TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL,
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX session_account_id_idx ON session (account_id);

-- +goose Down
DROP TABLE session;
