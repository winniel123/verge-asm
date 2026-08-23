-- +goose Up
-- Per-account message read-state (#327). The message table carries a single
-- global read_at column (20500_message.sql), but read-state is a per-account
-- fact: whether THIS operator has seen a message, not whether ANYONE has. With
-- one global column, MarkAllMessagesRead cleared the unread badge for every
-- account at once — a low-priv viewer could suppress the security notifications
-- an admin had not yet read. This join table makes read-state the (account,
-- message) pair it always should have been.
--
-- The old message.read_at column is deliberately LEFT IN PLACE, not dropped:
-- the read path (ListMessages / toMessageRow) still selects it, and dropping it
-- would break that reader in the same change. It is simply no longer written —
-- the mark handlers now write here — so it stays NULL for new rows and this
-- table is the source of truth for per-account read-state. Nothing is migrated:
-- existing rows carry no per-account read rows, so every message starts unread
-- per-account, which is the honest default (we cannot know who had read what
-- under the old global model).
CREATE TABLE message_read (
    -- The account that read the message. ON DELETE CASCADE so removing an account
    -- takes its read-marks with it.
    account_id BIGINT NOT NULL REFERENCES account (id) ON DELETE CASCADE,
    -- The message that was read. ON DELETE CASCADE so a deleted message leaves no
    -- orphan read-marks (messages are retention-only today, but the FK is honest).
    message_id BIGINT NOT NULL REFERENCES message (id) ON DELETE CASCADE,
    -- When this account first read this message. Set explicitly by the caller;
    -- defaulted to now() for safety. Idempotent at the query layer: a second mark
    -- leaves the first instant in place (ON CONFLICT DO NOTHING), since read-state
    -- is a fact about having seen it and does not move on a re-read.
    read_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_id, message_id)
);

-- The per-message read lookups (count-unread's NOT EXISTS, and the ON DELETE
-- CASCADE from message) probe by message_id; the composite PK leads with
-- account_id, so add the reverse index.
CREATE INDEX message_read_message ON message_read (message_id);

-- +goose Down
DROP TABLE message_read;
