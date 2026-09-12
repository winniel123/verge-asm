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
-- Existing rows backfill true, because a false would hold every already-pending message behind a
-- dispatch that nothing will ever mark. `goose.Up` runs in cmd/web and the dispatcher runs in
-- cmd/worker, so a worker mid-fan-out during this migration has its dispatch marked while it still
-- streams. That row then reads drained once its committed jobs finish, which is what the proxy this
-- column replaces already did (#847). The backfill restores no worse a reading than it removes.
ALTER TABLE dispatch ADD COLUMN fanout_complete BOOLEAN NOT NULL DEFAULT false;
UPDATE dispatch SET fanout_complete = true;

-- A crashed fan-out marks neither column, so a later tick of the same Scan marks this one and the
-- release bound passes over the row. It is a second column rather than a fifth `status` token
-- because ADR-0164 §1 rules that `status` carries the operator's disposition, and its #1523
-- amendment prices a machine-minted token: `stopped` and `terminated` each cancel jobs and mean a
-- person acted, and `dispatchOutcome` renders every token it names. Abandonment cancels nothing,
-- names no person, and moves no operator surface, so it is not a disposition and takes no token.
-- No backfill: an existing row is not abandoned.
ALTER TABLE dispatch ADD COLUMN fanout_abandoned BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE dispatch DROP COLUMN fanout_abandoned;
ALTER TABLE dispatch DROP COLUMN fanout_complete;
