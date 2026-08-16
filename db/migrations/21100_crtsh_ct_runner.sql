-- +goose Up
-- The crt.sh CT runner (ADR-0106). Certificate transparency is v1's flagship
-- keyless discovery source; ADR-0027 ruled it admits `Name`s on
-- `authority: inferred`, observes no facet and holds no timeline, and ADR-0096
-- ruled its `Citation` never ages. This ticket builds the covering execution
-- path #241 held open, in three parts: the sixth `Scan` that schedules the poll,
-- the store an admission lands in, and the instance-wide throttle ADR-0005
-- requires.

-- The `ct` Scan (ADR-0106) — the exchange, not the instrument (ADR-0084's
-- `dns`-not-`discovery` move). Worker-read, no port list, no vantage, and — its
-- source admitting without observing — no currency bound and no withdrawal power
-- (ADR-0096 §7): it schedules and gives Coverage a row, and bounds nothing. The
-- CHECK is widened here rather than in 18801 so the kind set travels with the
-- runner that introduces it.
ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct'));

-- Ships enabled at daily — the discovery cadence, matching `dns` (ADR-0106). The
-- Scan is the Declared schedule; whether CT actually runs is gated separately on
-- the `crtsh` source's enablement, so a disabled source leaves this Scan firing
-- over an empty scope (a legible zero-job state, like `zone` with no file).
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('ct', TRUE, 86400);

-- A CT admission, materialised (ADR-0027, ADR-0106). The estate is
-- observation-derived and CT produces no observation, so this is where "admits
-- without observing" lands: one row per `Name` a crt.sh Batch admitted, carrying
-- the Batch that admitted it (the `Citation` hop, ADR-0027) and the covering
-- name-scope Seed the chain terminates at. It is NOT membership — a Name becomes
-- a current member only when our own resolver resolves it (ADR-0096 §5); this row
-- records how it entered, never that it is here. No facet, no value, no timeline.
CREATE TABLE admitted_name (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT NOT NULL,
    source     TEXT NOT NULL,
    seed_id    BIGINT NOT NULL REFERENCES seed (id),
    batch_id   BIGINT NOT NULL REFERENCES batch (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The read behind provenance ("why is this name here"): the latest admission per
-- (name, source). A re-poll of an append-only source re-admits the same names, so
-- the current admission is the newest batch that named the subject — which is what
-- keeps the citing Batch retained under ADR-0041's second limb.
CREATE INDEX admitted_name_latest_idx ON admitted_name (name, source, id DESC);
CREATE INDEX admitted_name_batch_idx ON admitted_name (batch_id);

-- The instance-wide 5 req/min throttle (ADR-0005: per-source across the whole
-- instance, in Postgres, not worker memory — crt.sh cut its limits twice for
-- abuse). A single reservation row: each fetch atomically claims the next free
-- slot before it goes on the wire, so `--scale worker=N` cannot exceed the
-- ceiling. It sits outside every derivation — it changes only timing.
CREATE TABLE crtsh_throttle (
    id           BIGINT PRIMARY KEY,
    next_free_at TIMESTAMPTZ NOT NULL
);
INSERT INTO crtsh_throttle (id, next_free_at) VALUES (1, now());

-- +goose Down
DROP TABLE crtsh_throttle;
DROP TABLE admitted_name;
DELETE FROM scan WHERE kind = 'ct';
ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns'));
