-- +goose Up
-- A Proposal is a candidate address scope a *proposer* offers the operator (v1
-- spec §3.1/§6.4, ADR-0012). A proposer admits nothing, so what it yields is not
-- an Observation and never lands in the observation table: a registry lookup
-- returning ranges it believes the operator holds is read by nothing until the
-- operator confirms it into a Seed. Proposal is Declared and carries no timeline
-- — a record re-offered with different contents is a *new* Proposal, never an
-- existing one changed (CONTEXT.md, ADR-0012).
--
-- Proposals are produced only in answer to an operator act — searching the
-- org-name box — never on a cadence, so they never accumulate into a queue. One
-- such act is a proposer_lookup: it groups every candidate a single search
-- produced, which is the unit a *decline* operates over (ADR-0022: declining may
-- be done over a whole lookup at once).
CREATE TABLE proposer_lookup (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    query      TEXT NOT NULL,
    created_by BIGINT NOT NULL REFERENCES account (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One proposed address scope. It records *which kind of record produced it* — an
-- RIR delegation, or a compelled reassignment written by an upstream provider —
-- because those carry different caveats and the operator is the one judging them
-- (ADR-0012). The scope is the native cidr type so a later confirmation copies it
-- straight into a Seed.
--
-- status is the Proposal's whole lifecycle, and it is deliberately not a
-- timeline: a pending Proposal is read by nothing; confirming one is singular and
-- creates exactly one Seed, whose id is retained here as provenance so "why is
-- this here?" has an honest answer one hop above the Seed (ADR-0012); declining
-- may be done in bulk over a whole lookup (ADR-0022) and records the operator's
-- "not mine" without ever being re-offered.
CREATE TABLE proposal (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    lookup_id         BIGINT NOT NULL REFERENCES proposer_lookup (id),
    source_slug       TEXT NOT NULL,
    record_kind       TEXT NOT NULL CHECK (record_kind IN ('rir-delegation', 'compelled-reassignment')),
    address_cidr      CIDR NOT NULL,
    org_name          TEXT NOT NULL DEFAULT '',
    status            TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'confirmed', 'declined')),
    confirmed_seed_id BIGINT REFERENCES seed (id),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- The confirmed_seed_id link exists exactly when the Proposal is confirmed:
    -- a pending or declined Proposal became no Seed, a confirmed one became
    -- exactly one.
    CONSTRAINT proposal_confirm_shape CHECK (
        (status = 'confirmed' AND confirmed_seed_id IS NOT NULL)
        OR (status <> 'confirmed' AND confirmed_seed_id IS NULL)
    )
);

-- The pending set is the only set the Seeds screen reads, and it is read by
-- lookup; a partial index keeps that read cheap without indexing the confirmed
-- and declined tail the screen never lists.
CREATE INDEX proposal_pending ON proposal (lookup_id) WHERE status = 'pending';

-- +goose Down
DROP TABLE proposal;
DROP TABLE proposer_lookup;
