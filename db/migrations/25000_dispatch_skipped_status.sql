-- +goose Up
-- A cadence-lag skip (ADR-0137 §4) commits its Dispatch row and enqueues nothing: rolling the
-- row back would leave the window unclaimed and let a later poll inside it dispatch, which is the
-- deferral §4 rules out. Until now that row carried status 'fanned-out', so a skip was
-- indistinguishable from a dispatch that genuinely found nothing and from one whose jobs the
-- ADR-0041 retention sweep retired. A record that states something the run did not do is the
-- wrong-record failure the whole #1092 thread objects to.
--
-- The Dispatch gains a fourth status, 'skipped': the tick was claimed so nothing else may claim it,
-- and no job was enqueued because an earlier dispatch of the same Scan had not drained. It is not a
-- terminal disposition in the sense 'stopped' and 'terminated' are — no operator acted, and no work
-- was cancelled — it is the recorded absence of a run. The drift engine never reads dispatch or
-- queue_job (ADR-0041), so this moves no estate truth.
ALTER TABLE dispatch DROP CONSTRAINT IF EXISTS dispatch_status_check;
ALTER TABLE dispatch
    ADD CONSTRAINT dispatch_status_check
    CHECK (status IN ('fanned-out', 'stopped', 'terminated', 'skipped'));

-- +goose Down
UPDATE dispatch SET status = 'fanned-out' WHERE status = 'skipped';
ALTER TABLE dispatch DROP CONSTRAINT IF EXISTS dispatch_status_check;
ALTER TABLE dispatch
    ADD CONSTRAINT dispatch_status_check
    CHECK (status IN ('fanned-out', 'stopped', 'terminated'));
