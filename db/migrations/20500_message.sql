-- +goose Up
-- A Message is one firing of one cause, computed once at the cause and never
-- recomputed (CONTEXT.md `Message`, v1 spec §5.3, ADR-0064). It is Operational:
-- it records that the operator was told, never what is true of the estate, so
-- the comparison path may never read this table and nothing is ever concluded by
-- comparing two rows. The fact itself is in the timelines; if the two ever
-- disagree the timeline wins and the message is still a true record of what we
-- said — which is what stops a stored message being a second representation of
-- one fact, and keeps `Finding` from returning as a diffed message log.
--
-- Every message is written and rendered unconditionally: the store is not a
-- `Channel`, has no configuration, cannot be disabled and cannot fail. Delivery
-- (the outbound POST and its per-Channel outcome) is a separate corpus and a
-- separate ticket (#207); it is not modelled here.
CREATE TABLE message (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- The cause is the model's own closed union of four — the estate's own object
    -- (drift), us (aperture), the operator's own declared input (declared-input),
    -- or nothing at all where only a threshold was crossed (threshold). It is a
    -- field the operator reads; the router never reads it (ADR-0091).
    cause        TEXT NOT NULL,
    -- The routing class is the four causes with two merged: aperture and
    -- declared-input both collapse to coverage. It is a property of the firing,
    -- so a clock-reading rule that finds its span moved is stored as drift.
    class        TEXT NOT NULL,
    -- The subject or scope the message fired at: subject_kind names what fired_at
    -- keys ("name"/"address"/"service"/"endpoint" for an object, "source" for a
    -- declared-input Source, "seed" for an aperture Seed), so the panel can build
    -- the right per-mover drill-down. The unit of alerting is the message and
    -- never the affected subject, so this is one key, not a set.
    subject_kind TEXT NOT NULL,
    fired_at     TEXT NOT NULL,
    -- The instant of the cause, read from the fold at construction and frozen. It
    -- is never re-derived — re-deriving would reach back across a `Break`.
    instant      TIMESTAMPTZ NOT NULL,
    -- The census payload where the firing has one (a flagship or membership root):
    -- a flat, enumerable list of the rows that opened beneath the fired-at
    -- subject. NULL where the firing carries only a count (a narrowing) or none.
    census       JSONB,
    -- The rendered sentence, computed once at the cause. It carries no valence
    -- word and no severity (ADR-0064); the store keeps it verbatim and the read
    -- path never recomputes it.
    headline     TEXT NOT NULL,
    -- Read-state is the one piece of operator state a message holds — NULL means
    -- unread. It is not part of the fact and never reaches the comparison path.
    read_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT message_cause CHECK (cause IN ('drift', 'aperture', 'declared-input', 'threshold')),
    CONSTRAINT message_class CHECK (class IN ('drift', 'coverage', 'clock'))
);

-- The panel opens an unbounded list newest-first and carries an unread count on
-- every screen; both read this partial index rather than scanning the table.
CREATE INDEX message_unread ON message (id DESC) WHERE read_at IS NULL;

-- +goose Down
DROP TABLE message;
