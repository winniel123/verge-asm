-- +goose Up
-- An index for ScanHasCompletedBatch — *has a Scan of this kind ever completed a
-- Batch?* (ticket #1018).
--
-- That read used to run on an EMPTY measurement store alone, so it was rare enough to
-- cost nothing. #1018 made the `edge-fanout` errored floor PER LIMB, and a non-empty
-- store no longer answers it: the two limbs share one store, so a declaration-limb row
-- says nothing about whether the extension limb was measured. The read is therefore
-- unconditional, on every hot dispatch — inside the transaction holding the per-scan
-- advisory lock — and on every render of the Scope and Coverage screens.
--
-- `batch` holds ONE ROW PER QUEUE JOB OUTCOME and grows with probing, so a sequential
-- scan there is not a fixed cost. It is worst exactly where the answer matters most: an
-- install whose `edge-fanout` Scan has NOT completed a Batch has no row to stop at, so
-- the EXISTS reads the whole table every time — and that is the install whose candidates
-- the floor must hold rather than reach.
--
-- PARTIAL on the outcome, so the index holds completed Batches alone. A dead-lettered
-- Batch does not answer this question — it is the job failing, and the tick retries —
-- and leaving it out keeps the index to the rows the predicate can return.
CREATE INDEX batch_kind_completed_idx ON batch (kind) WHERE outcome = 'completed';

-- +goose Down
DROP INDEX batch_kind_completed_idx;
