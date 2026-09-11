-- +goose Up
-- The instant the release poll read this fold's re-point residue (ADR-0026 §2, ADR-1806 §2).
--
-- ADR-0026 §2 fires a message where a resolution move opens an Endpoint no membership message
-- covers. The Endpoints open in a later hot fold, so the move's own fold counts none and the
-- message never fires (#1774). The poll re-derives the move from the two adjacent spans, which
-- are durable, and reads the residue once the admitting tier has drained.
--
-- The re-point path takes no census_pending_after_batch row. ADR-1806 §2 gives membership one
-- because its root determination is fold-local; a move's predicate reads two spans and nothing
-- else. What the path still owes is a once-only bound: the spans stay true after the message
-- fires, and the poll runs every minute (ADR-1806 §8). This column is that bound. The guarded
-- UPDATE is the claim, so a second pass takes no row and writes no second message.
--
-- The unit is the fold and not the move, because the fold is the unit the move's own predicate
-- reads: one batch's moves share the candidate-address set that decides which of them the
-- Address root already covers (internal/queue/repoint.go).
ALTER TABLE batch
    ADD COLUMN repoint_settled_at TIMESTAMPTZ;

-- A move that closed before this decision shipped is settled by the migration. Announcing the
-- history on the first poll pass would page the operator for every move the estate ever folded.
--
-- The backfill runs BEFORE the index. Built first, the index would cover every row, and this
-- UPDATE would then churn every entry back out of it.
UPDATE batch SET repoint_settled_at = now();

-- Every batch is a candidate, so the scan must be proportional to what is unsettled and never to
-- the table. One queue job is one Batch (18803_measurement_batch.sql), so the table is the
-- largest operational record the estate keeps.
CREATE INDEX batch_repoint_unsettled_idx ON batch (id) WHERE repoint_settled_at IS NULL;

-- +goose Down
DROP INDEX batch_repoint_unsettled_idx;

ALTER TABLE batch
    DROP COLUMN repoint_settled_at;
