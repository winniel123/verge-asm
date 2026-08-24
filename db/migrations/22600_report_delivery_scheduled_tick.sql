-- +goose Up
-- The on-cadence report dispatcher (#502/T3) reuses the report_delivery receipt
-- (migration 22500) as its own dispatch record: a scheduled run's receipt already
-- IS the one-run record, so there is deliberately no separate report_dispatch
-- table. What a scheduled run needs beyond a manual one is an idempotency key — the
-- tick it was dispatched for — so a second poll inside the same cadence window
-- conflicts instead of stamping a second receipt.
--
-- scheduled_tick is the cadence boundary the run was dispatched for, floored to the
-- window (internal/report scheduledTick). It is NULLABLE: a manual "Run now" receipt
-- leaves it NULL — a manual run is keyed to the instant the operator asked and never
-- contends on a tick, so it must not collide with a scheduled run or with another
-- manual run of the same schedule.
--
-- The partial-unique (schedule_id, scheduled_tick) WHERE scheduled_tick IS NOT NULL
-- is the on-cadence idempotency backstop. It mirrors the queue dispatcher's unique
-- (scan, scheduled_time): the first poll in a window inserts and wins the claim; a
-- later poll conflicts and returns no row (a recorded skip, not a double-run). The
-- partial predicate keeps every NULL-tick manual receipt out of the index, so manual
-- runs stay unconstrained while scheduled runs are one-per-(schedule, tick).
ALTER TABLE report_delivery ADD COLUMN scheduled_tick TIMESTAMPTZ;

CREATE UNIQUE INDEX report_delivery_scheduled_tick
    ON report_delivery (schedule_id, scheduled_tick)
    WHERE scheduled_tick IS NOT NULL;

-- +goose Down
DROP INDEX report_delivery_scheduled_tick;
ALTER TABLE report_delivery DROP COLUMN scheduled_tick;
