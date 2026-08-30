-- +goose Up
-- Per-source CT throttle (spec docs/spec/ct-source-replacement.md §2.5, map #854).
-- The `ct` source path prepares to carry more than one source with no user-visible
-- change. The single-row crtsh_throttle — one instance-wide crt.sh reservation
-- (ADR-0005, migration 21100) — becomes a reservation table keyed by the source
-- slug, so each CT source reserves on its own interval without a new table. crt.sh
-- keeps its behaviour: one row, the same 12 s interval the caller supplies.
CREATE TABLE ct_throttle (
    source       TEXT PRIMARY KEY,
    next_free_at TIMESTAMPTZ NOT NULL
);

-- Carry crt.sh's reservation across so the first reserve after this migration
-- behaves exactly as before. crt.sh is the only active source, so its row is the
-- only one seeded here; a second source seeds its own row when it ships.
INSERT INTO ct_throttle (source, next_free_at)
VALUES ('crtsh', COALESCE((SELECT next_free_at FROM crtsh_throttle WHERE id = 1), now()));

DROP TABLE crtsh_throttle;

-- +goose Down
CREATE TABLE crtsh_throttle (
    id           BIGINT PRIMARY KEY,
    next_free_at TIMESTAMPTZ NOT NULL
);
INSERT INTO crtsh_throttle (id, next_free_at)
VALUES (1, COALESCE((SELECT next_free_at FROM ct_throttle WHERE source = 'crtsh'), now()));

DROP TABLE ct_throttle;
