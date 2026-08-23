-- +goose Up
-- A Span cites the Batch that folded it (ADR-0111, #288), on each side it has: the
-- Batch whose fold OPENED the span, and the Batch whose fold CLOSED it. This is the
-- link the estate-wide drift feed groups by — the /drift screen's whole composition
-- is "what moved, grouped by batch", and a transition is derived on read from two
-- adjacent spans (ADR-0007), so the only thing the store lacked was which Batch
-- caused a given open or close.
--
-- Both reference `batch`, NEVER `dispatch`. ADR-0041 places `Batch` in corpus 1 (the
-- comparison path may read it) and `Dispatch` in corpus 2 (it may not) — precisely so
-- the operational grouping cannot be reached from the comparison path. A span citing
-- its batch is the corpus-1 citation ADR-0041 permits; a span citing a dispatch would
-- be the corpus-2 reach it forbids.
--
-- Both are NULLABLE:
--   - Spans written before this migration carry no batch id — the corpus is never
--     rewritten (ADR-0041), so they stay honestly un-attributable and the feed groups
--     only what it can.
--   - An open span has no closing batch yet.
--   - A withdrawal closure is not a batch fold — whether a subject left the estate is a
--     cross-class composition (internal/estate.WithdrawnCrossClass, ADR-0087), not one
--     batch's outcome — so a withdrawal-closed span may legitimately carry no
--     closed_batch_id.
ALTER TABLE span
    ADD COLUMN opened_batch_id BIGINT REFERENCES batch (id),
    ADD COLUMN closed_batch_id BIGINT REFERENCES batch (id);

-- The feed reads recent events by the batch that caused them, so an index over each
-- citation keeps "the spans this batch opened / closed" cheap. Partial on NOT NULL:
-- the overwhelming majority of the closed column is NULL (every open span, plus every
-- pre-migration row), and a partial index carries only the rows the feed actually
-- joins on.
CREATE INDEX span_opened_batch_idx ON span (opened_batch_id) WHERE opened_batch_id IS NOT NULL;
CREATE INDEX span_closed_batch_idx ON span (closed_batch_id) WHERE closed_batch_id IS NOT NULL;

-- +goose Down
DROP INDEX span_closed_batch_idx;
DROP INDEX span_opened_batch_idx;
ALTER TABLE span DROP COLUMN closed_batch_id, DROP COLUMN opened_batch_id;
