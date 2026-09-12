-- +goose Up
-- ADR-1806 §3 defines the release bound as two halves: the Dispatch fanned out, and every
-- queue_job it enqueued reached a terminal state. The record held only the second.
--
-- A streamed fan-out (hot, cold, edge-fanout) commits its dispatch row in claimDispatch and
-- streams its jobs in later chunks, so `status = 'fanned-out'` is written before the fan-out
-- finishes and cannot answer the first half. #1816 used a non-empty job set as the proxy.
-- #1851 records why that proxy fails: a tier that is enabled and admits nothing finishes its
-- fan-out with no job at all, and is drained rather than pending (CONTEXT.md, `Drained`).
--
-- The column carries the first half and nothing else. It holds no instant, because a duration
-- in a release predicate is what ADR-1806 §7 and #27 both refuse.
--
-- Existing rows backfill true. Their fan-out is not in flight, and a false would hold every
-- already-pending message behind a dispatch that nothing will ever mark.
ALTER TABLE dispatch ADD COLUMN fanout_complete BOOLEAN NOT NULL DEFAULT false;
UPDATE dispatch SET fanout_complete = true;

-- +goose Down
ALTER TABLE dispatch DROP COLUMN fanout_complete;
