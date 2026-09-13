-- +goose Up
-- The last batch of one dispatch, read once per held message on every release poll (ADR-1870 §3).
--
-- ADR-1870 bounds the census read at the top. The bound is the last batch of the hot dispatch
-- whose drain released the row, which ListReleasableHeldMessages reads as
-- max(batch.id) WHERE batch.dispatch_id = the chosen dispatch.
--
-- batch carried no index on dispatch_id. batch_scan_created_idx leads on scan_id and answers
-- nothing here, so that read is a sequential scan of batch, once per held row, once a minute.
-- ADR-1870 exists to cut what a census costs, and an unindexed scan would hand the cost back.
--
-- DESC puts the greatest id first, so the planner takes the max from the leading entry of the
-- matching group rather than scanning the group.
CREATE INDEX batch_dispatch_id_idx ON batch (dispatch_id, id DESC) WHERE dispatch_id IS NOT NULL;

-- +goose Down
DROP INDEX batch_dispatch_id_idx;
