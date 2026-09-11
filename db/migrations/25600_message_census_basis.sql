-- +goose Up
-- The basis a held census is computed from, frozen at the cause (ADR-1806 §4).
--
-- A held row's census is computed later, by the release poll, and "beneath the root" must mean
-- what the fold saw. Live resolution at release time would fold a second move's subjects into the
-- first move's message, under the first move's instant, so the poll may never read it.
--
-- The value is the root span's own value at the root's batch, with the root it belongs to:
-- {"root_kind","root_key","root_value"}. The root is carried here and not read from subject_kind
-- and fired_at, because a revealed firing fires at the Seed whose scope moved and names no root
-- at all (v1 spec §5.3, ADR-0031).
--
-- The span table cannot answer this later. A span carries closed_batch_id alone, so the span a
-- batch opened is not addressable by that batch.
ALTER TABLE message
    ADD COLUMN census_basis JSONB;

-- A held row owes its basis. A released row keeps the basis its census was computed from.
ALTER TABLE message
    ADD CONSTRAINT message_census_pending_has_basis
    CHECK (census_pending_after_batch IS NULL OR census_basis IS NOT NULL);

-- +goose Down
ALTER TABLE message
    DROP CONSTRAINT message_census_pending_has_basis;

ALTER TABLE message
    DROP COLUMN census_basis;
