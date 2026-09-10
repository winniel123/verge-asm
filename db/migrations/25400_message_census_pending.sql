-- +goose Up
-- ADR-1806 amends two sentences of 20500_message.sql, and a migration is immutable, so the
-- amendment is stated here rather than by editing that file.
--
-- First: a message whose firing carries a census is still written once at its cause, but the
-- census, and the census clause of the headline it produces, are computed later — when the
-- tier that admits the entering sub-tree has drained. A fold is one job with one Kind, so the
-- Service and Endpoint spans a census counts open in a later hot fold and a census read in the
-- root's own fold is empty (#1774).
--
-- Second: such a row is not written and rendered unconditionally. While this column is non-NULL
-- the row is held: it reaches no panel, no bell and no unread badge, and mark-all-read passes
-- over it, so releasing it later still shows it as unread. Nothing reaches the operator until
-- the census is real.
--
-- The census column cannot carry the mark. NULL there already means "the firing carries only a
-- count, or none", so it cannot also mean pending, and a released empty census would be
-- indistinguishable from a held one (ADR-1806 §2, §7).
--
-- The value is the root's own batch. The release poll releases the row once the first hot
-- dispatch that fanned out after that batch has drained (ADR-1806 §3). No writer sets the column
-- yet, so after this migration no row is held and operator behaviour is unchanged.
ALTER TABLE message
    ADD COLUMN census_pending_after_batch BIGINT REFERENCES batch (id);

-- +goose Down
ALTER TABLE message
    DROP COLUMN census_pending_after_batch;
