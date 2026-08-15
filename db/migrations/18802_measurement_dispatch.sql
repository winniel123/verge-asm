-- +goose Up
-- A Dispatch groups the progress of one fan-out of a Scan (v1 spec §4.1). It is
-- Operational — it carries no observations and the drift engine never reads it.
-- It fires under a Postgres advisory lock and is idempotent on
-- (scan, scheduled_time): the UNIQUE constraint is what makes an overlapping
-- tick a recorded `skipped` row rather than a second concurrent fan-out. Missed
-- ticks are not caught up — you cannot measure the past.
CREATE TABLE dispatch (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scan_id        BIGINT NOT NULL REFERENCES scan (id),
    scheduled_time TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL CHECK (status IN ('fanned-out', 'skipped')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dispatch_tick_key UNIQUE (scan_id, scheduled_time)
);

-- +goose Down
DROP TABLE dispatch;
