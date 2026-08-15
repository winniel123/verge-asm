-- +goose Up
-- A Vantage is ONE network position observations are made from (CONTEXT.md
-- `Vantage`). Its measurement identity is mandatory: a name, a class re-verified
-- every batch, and the recursive resolver it resolves through — the resolver is
-- part of that position and therefore part of the term's identity (ADR-0070), so
-- it lives on the row and never as a leaf parameter.
--
-- A prober connection is OPTIONAL provisioning detail layered onto that same
-- position (ADR-0103). Provisioning a prober is the act that DECLARES "this
-- vantage is on the internet" (#124): there is no network_position field and no
-- setup-wizard step, the intent is carried by the act of provisioning. The
-- prober columns (host, port, username, availability, public_key, host_key,
-- created_by) are therefore NULL for a resolver-only vantage that has no prober,
-- such as the shipped `local` vantage, and are set together when an operator
-- provisions an endpoint.
--
-- The operator supplies host, port and a non-root username. The instance
-- generates the SSH keypair on a worker-only volume and only the public half
-- ever leaves it: public_key holds that public half (rendered on web), and the
-- private half is never in Postgres at all. host_key is the SSH host key pinned
-- on first successful connection — trust-on-first-use — and a later mismatch is
-- a hard failure that marks the vantage unavailable rather than silently
-- re-trusting.
CREATE TABLE vantage (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- Measurement identity (mandatory on every Vantage).
    name         TEXT NOT NULL UNIQUE,
    class        TEXT NOT NULL DEFAULT 'unverified'
                   CHECK (class IN ('internet', 'internal', 'unverified')),
    -- The recursive resolver; ships blank for a freshly provisioned prober,
    -- which the operator then sets. NOT NULL so the dispatch reader never has to
    -- reason about a missing resolver — an unset one is the empty string.
    resolver     TEXT NOT NULL DEFAULT '',

    -- Optional prober-connection detail. All NULL together for a resolver-only
    -- vantage with no prober; all set together when an endpoint is provisioned.
    host         TEXT,
    port         INTEGER CHECK (port IS NULL OR port BETWEEN 1 AND 65535),
    username     TEXT,

    -- Availability is the one Derived property on this Declared term (CONTEXT.md),
    -- and only a provisioned prober has one (NULL for the resolver-only vantage):
    --   'pending'     — provisioned, no successful connection has pinned a host key yet
    --   'available'   — host key pinned, the position is reachable
    --   'unavailable' — a pinned host key later mismatched, or the position went unreachable
    availability TEXT CHECK (availability IS NULL
                   OR availability IN ('pending', 'available', 'unavailable')),

    -- The public half of the instance-generated SSH keypair, in authorized_keys
    -- form. NULL until the worker has generated the pair on its own volume. The
    -- private half never appears in this database.
    public_key   TEXT,

    -- The SSH host key pinned on first successful connection (trust-on-first-use),
    -- in known_hosts form. NULL until first connect.
    host_key     TEXT,

    -- The admin who provisioned the prober; NULL for a shipped/resolver-only
    -- vantage that no operator declared.
    created_by   BIGINT REFERENCES account (id),

    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A prober endpoint is provisioned once. Two vantages differing only in host key
-- or availability are the same declared position, so uniqueness is on the
-- (host, port, username) the operator dials. NULLs (resolver-only vantages) are
-- distinct under this index, so any number of prober-less vantages may coexist.
CREATE UNIQUE INDEX vantage_endpoint_key ON vantage (host, port, username);

-- +goose Down
DROP TABLE vantage;
