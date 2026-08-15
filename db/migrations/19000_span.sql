-- +goose Up
-- A Span is one period a (subject, facet, discriminator, vantage, source)
-- timeline held a single value — opened, current, closed (v1 spec §5.1,
-- ADR-0007). It is the drift engine's one stored object: a Transition is the
-- adjacency between two consecutive spans and a Break is the boundary between two
-- under differing Derivation vectors, and BOTH are derived on read, never stored
-- (ADR-0007, ADR-0008) — there is no transition table and no break table by
-- design, because storing either is a second representation of one fact.
--
-- Scoped to the two facets that exist so far (`resolution`, `dns-record`); the
-- machinery is facet-agnostic and later facets reuse this table unchanged.
--
-- The Span corpus is NEVER compacted or deleted (ADR-0041): a span is written
-- when a value MOVES, so the corpus is proportional to drift rather than to time.
-- There is deliberately no delete/compaction query against this table anywhere,
-- and the retention path (20900) is barred from reaching it. Current state is a
-- lookup — the one open span per timeline — rather than a fold.
CREATE TABLE span (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_kind   TEXT NOT NULL CHECK (subject_kind IN ('name', 'address', 'service', 'endpoint')),
    subject_key    TEXT NOT NULL,
    facet          TEXT NOT NULL CHECK (facet IN ('resolution', 'dns-record')),
    discriminator  TEXT NOT NULL DEFAULT '',
    vantage_id     BIGINT REFERENCES vantage (id),
    source         TEXT NOT NULL DEFAULT 'resolver',
    -- The canonical value the span held, compared structurally; a hash is only an
    -- index (ADR-0007). is_gap marks a span holding NO value — the period over
    -- which the system could not say — which never withdraws a subject.
    value          JSONB NOT NULL,
    is_gap         BOOLEAN NOT NULL DEFAULT FALSE,
    -- The flattened, deduped vector of Derivation component versions the span was
    -- produced under: a JSON array of {"leaf","version"} (ADR-0008). Two spans
    -- compare only where these are equal; the Break naming the moved leaf is read
    -- from the two arrays and never stored.
    derivation     JSONB NOT NULL,
    opened_at      TIMESTAMPTZ NOT NULL,
    -- NULL closed_at is the current span. A closure_reason is carried ONLY by a
    -- withdrawal's closure — the closed union of three grounds (ADR-0087); an
    -- ordinary value move records none (the next span is the fact) and a version
    -- change records none (the Break is derived from the two vectors).
    closed_at      TIMESTAMPTZ,
    closure_reason TEXT CHECK (closure_reason IN ('measured-absent', 'uncited', 'descoped')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- A reason may sit only on a closed span; an open span has drawn no boundary.
    CONSTRAINT span_reason_needs_closure CHECK (closure_reason IS NULL OR closed_at IS NOT NULL)
);

-- At most one open span per timeline: current state is that one row, read as a
-- lookup. The partial unique index is what makes "the open span is the current
-- state" a structural guarantee rather than a query convention. NULLS NOT
-- DISTINCT so the shipped resolver position — which carries no vantage row, hence
-- a NULL vantage_id — is still held to one open span (PostgreSQL 16).
CREATE UNIQUE INDEX span_open_timeline_idx
    ON span (subject_key, facet, discriminator, vantage_id, source)
    NULLS NOT DISTINCT
    WHERE closed_at IS NULL;

-- Timeline read: a subject's spans in facet / discriminator / vantage / source
-- order, oldest first, for the drill-down's current + closed rendering.
CREATE INDEX span_subject_idx
    ON span (subject_kind, subject_key, facet, discriminator, vantage_id, source, opened_at);

-- +goose Down
DROP TABLE span;
