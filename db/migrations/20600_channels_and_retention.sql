-- +goose Up
-- A Channel is the operator's Declared assertion of where Messages go (v1 spec
-- §6.1, CONTEXT.md "Channel", ADR-0039): an absolute https URL, an optional
-- write-only signing secret, and the subset of the three routing classes it
-- receives. It is input on Proposal's own layer — nothing in the comparison path
-- reads it — and none ships configured. This migration lands the store only; the
-- outbound POST and the Delivery record are ticket 27's, so there is no delivery
-- state here.
CREATE TABLE channel (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url            TEXT NOT NULL,
    -- The HMAC signing secret. Nullable (a channel may sign nothing) and
    -- write-only at the interface: it is set, replaced or cleared through the
    -- UI and never rendered back, on the footing recovery codes set (#11). No
    -- render query selects this column; ListChannels exposes only whether it is
    -- present.
    secret         TEXT,
    -- Routing is by class and nothing finer (ADR-0091). Each class is a boolean
    -- rather than a set column so the database can enforce that at least one is
    -- carried: a channel routing no class would receive nothing.
    route_drift    BOOLEAN NOT NULL DEFAULT true,
    route_coverage BOOLEAN NOT NULL DEFAULT true,
    route_clock    BOOLEAN NOT NULL DEFAULT true,
    -- Disabling is not a predicate change (notification-channels.md §2): a
    -- disabled channel still exists and still routes the same classes, it just
    -- carries nothing until re-enabled.
    enabled        BOOLEAN NOT NULL DEFAULT true,
    created_by     BIGINT NOT NULL REFERENCES account (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT channel_has_class CHECK (route_drift OR route_coverage OR route_clock)
);

-- The two retention dials (v1 spec §4.6), a single operator-global row. The dials
-- collapse to one number each (ADR-0094); the query never does, and neither dial
-- is read by anything in v1 — this ticket persists the operator's chosen values
-- and the enforcing floors land with tickets 28/29.
--
--   observation_currency_days  the observation-currency floor, an age in days. A
--                              row is retained while inside EITHER its own bound
--                              OR this dial, whichever is longer.
--   dispatch_cadence_multiple  the Dispatch floor, stated as a multiple of the
--                              slowest enabled Scan's cadence, not a day count.
--
-- Both floor at zero for now (zero == no operator floor). The real floors — the
-- tightest bound in force for observations, k cadences of the slowest enabled
-- Scan for Dispatch — are validated once tickets 28/29 define them.
CREATE TABLE retention_settings (
    -- A single-row table: the boolean primary key can only ever hold one true
    -- row, so the dials are global and there is exactly one to read.
    id                        BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    observation_currency_days BIGINT NOT NULL DEFAULT 0 CHECK (observation_currency_days >= 0),
    dispatch_cadence_multiple BIGINT NOT NULL DEFAULT 0 CHECK (dispatch_cadence_multiple >= 0),
    updated_by                BIGINT REFERENCES account (id),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO retention_settings (id) VALUES (true);

-- +goose Down
DROP TABLE retention_settings;
DROP TABLE channel;
