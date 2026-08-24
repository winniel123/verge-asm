-- +goose Up
-- A signal_instance is the persistent identity of one raised signal — a
-- `(signal-name, subject)` pair the `Signal` engine has placed under `fired`
-- (P0.1; design-system/examples/console/SignalData.jsx). The census itself is
-- always re-derived live from the Derived corpus and is never stored; what this
-- table adds is the two things a per-instance row must carry that a pure
-- re-derivation cannot reconstruct: a STABLE, mintable id and the instant the
-- pair was FIRST seen firing.
--
-- Only identity + first-seen persist here. Everything else the SignalData.jsx row
-- shows is derived on read: the severity is the rule's (assigned per rule in
-- internal/signal, not stored), the last-seen instant is the current derivation
-- instant (the pair is confirmed firing now), and asset / ip / port fall out of
-- the subject key. This is the ADR-0092 discipline the `annotation` table already
-- follows — keyed on the exact pair, it never travels: a redeploy onto a new
-- subject mints a new instance rather than carrying an old id onto a different
-- fact, and a returning pair is the same row (its first-seen and id preserved).
--
-- The id is the mint. The display id the console renders — `SIG-####` — is
-- formatted from this identity by the web layer, so it is stable for the life of
-- the pair and never reused. It starts at 1000 so the first minted signal reads
-- SIG-1000 rather than a one-or-two-digit id.
CREATE TABLE signal_instance (
    id          BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 1000) PRIMARY KEY,
    signal_name TEXT NOT NULL,
    subject_key TEXT NOT NULL,
    first_seen  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keyed on the exact `(signal-name, subject)` pair, minted once. The unique index
-- makes the mint idempotent: re-deriving the same fired pair on every page load
-- inserts nothing new (ON CONFLICT DO NOTHING) and preserves the original id and
-- first-seen, so a signal that has been firing for a week keeps reporting when it
-- was first raised, not when it was last rendered.
CREATE UNIQUE INDEX signal_instance_pair_key ON signal_instance (signal_name, subject_key);

-- +goose Down
DROP TABLE signal_instance;
