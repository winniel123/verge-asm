-- +goose Up
-- A Dispatch groups the progress of one fan-out of a Scan (v1 spec §4.1). It is
-- Operational — it carries no observations and the drift engine never reads it.
-- It fires under a Postgres advisory lock and is idempotent on
-- (scan, scheduled_time): the UNIQUE constraint admits exactly ONE Dispatch per
-- tick, so an overlapping fan-out finds the tick already dispatched and is a
-- no-op. The overlap is "recorded" by that one fanned-out row — there is no
-- second `skipped` row, and the unique key makes one structurally impossible, so
-- 'fanned-out' is the only status the column ever holds. Missed ticks are not
-- caught up — you cannot measure the past.
CREATE TABLE dispatch (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scan_id        BIGINT NOT NULL REFERENCES scan (id),
    scheduled_time TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('fanned-out')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dispatch_tick_key UNIQUE (scan_id, scheduled_time)
);

-- +goose Down
DROP TABLE dispatch;
