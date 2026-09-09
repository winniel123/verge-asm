-- +goose Up
-- What this install's last attempt to a source did. The record is Operational
-- (ADR-0223 §1): it states what the system did, it varies per install, and no
-- session authors it. It is not the Declared enablement override, which is why
-- it is not a column on source_state — that row exists only where the operator
-- toggled a slug, so an untouched source has nowhere to write, and holding both
-- layers in one row is the confusion ADR-0223 removes.
--
-- The query path is the only writer (ADR-0223 §4). Nothing probes a source on a
-- schedule, because a probe sends a request to a third party for a source the
-- operator never enabled, which ADR-0003 governs and no operator authorised. A
-- source nobody enables therefore holds no row, and the absence reads as
-- "never attempted" and never as healthy.
--
-- The row holds the last outcome, its instant, and the consecutive-failure
-- count, and nothing else. A derived healthy/degraded/dead state is refused:
-- nobody has picked a threshold, and a derived word would state a conclusion
-- about the world from a record of our own requests. The operator draws the
-- conclusion. No value here may move source_state: an Operational record never
-- overwrites a Declared consent state (ADR-0223 §4, ADR-0003).
--
-- Keying on the source's stable slug, never a display name, keeps the catalogue
-- free to move a label without stranding a row (18500_source_state.sql).
CREATE TABLE source_health (
    slug                 TEXT PRIMARY KEY,
    last_outcome         TEXT NOT NULL CHECK (last_outcome IN ('ok', 'error')),
    last_attempt_at      TIMESTAMPTZ NOT NULL,
    consecutive_failures INTEGER NOT NULL
);

-- +goose Down
DROP TABLE source_health;
