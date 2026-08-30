-- +goose Up
-- The `ct-tail` Scan and its per-log cursor (spec docs/spec/ct-source-replacement.md
-- §4, map #854). CT-logs-direct's drift tail watches new issuance for names the
-- operator already knows. It is a SECOND CT Scan beside the bulk `ct` poll (ADR-0106):
-- it admits the same way — an `admitted_name` row on `authority: inferred` citing its
-- Batch (ADR-0027) — but it reads the logs directly, forward-only, and fans out
-- per-log rather than per name-scope Seed. It creates no facet, no Signal and no
-- timeline; its drift finding is an ephemeral per-job event, never a durable row (§4.1).

-- The kind CHECK is a closed union grown one member at a time (18801, widened for
-- `ct` in 21100 and `http-identity` in 23400). It is widened here so the kind set
-- travels with the dispatch that introduces it.
ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct', 'http-identity', 'ct-tail'));

-- Ships ENABLED (the Declared schedule) but its source ships OFF: `ct-tail` fans out
-- on cadence yet enqueues no job until the operator enables the `ct-tail` source
-- (DefaultOn: false, gated by source_state exactly like crtsh — §4.4). This mirrors
-- the `ct`/`crtsh` split: the Scan carries the cadence, the source toggle carries
-- consent. Cadence is a MEASURED BAR and an operator dial (§4.4): 300 s keeps the
-- per-poll forward delta inside one bounded fetch window on an unbusy log; a busy log
-- may lag and catch up over later polls. Moving it moves no version and Breaks nothing.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('ct-tail', TRUE, 300);

-- The tail's per-log forward cursor — the system's first durable per-target cursor
-- (§4.2). One row per CT log the tail follows, created lazily on the log's first poll.
-- `tree_size` is the last position read (S_last); the next poll reads only entries at
-- or beyond it, so the tail never backfills history (the §4 design invariant).
-- `signed_head` holds the last STH / checkpoint the log signed, so a later poll can
-- prove append-only continuity — running the proof is opportunistic, not mandatory
-- per poll (§4.4). No row migrates; all of this state is new.
CREATE TABLE ct_log_cursor (
    log_id      TEXT PRIMARY KEY,
    tree_size   BIGINT NOT NULL,
    signed_head BYTEA NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ct_log_cursor;

DELETE FROM scan WHERE kind = 'ct-tail';

ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct', 'http-identity'));
