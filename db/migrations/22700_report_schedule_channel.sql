-- +goose Up
-- A report_schedule's declared delivery destination is a Channel (P0.6c/T7, #508,
-- collision #17 ruled). channel_id binds the schedule to the signed-HTTPS Channel
-- that receives its ready-message; NULL is the default and means download-only —
-- the run generates the artifact in-instance and no notification leaves.
--
-- The Channel receives a LINK-ONLY ready-message: the report name, the run's period,
-- and a session-authed link to the in-instance artifact — never the estate. The
-- report body never leaves the instance (ADR-0039 stands). This binding supersedes
-- the free-text delivery_target as the destination: delivery_target stays in place
-- (migrations are append-only) but is written empty and no longer read as the target.
--
-- ON DELETE is left at the default (RESTRICT): a Channel an operator still routes a
-- schedule to cannot be deleted out from under it, matching the delivery table's own
-- channel_id reference. Editing a schedule to download-only (NULL) or another Channel
-- is the way to release the binding.
ALTER TABLE report_schedule
    ADD COLUMN channel_id BIGINT REFERENCES channel (id);

-- +goose Down
ALTER TABLE report_schedule
    DROP COLUMN channel_id;
