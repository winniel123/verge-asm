-- +goose Up
-- A Vantage is a network position observations are made from, declared as
-- intent and re-verified every batch (CONTEXT.md, v1 spec §4.2). Provisioning a
-- prober is the act that DECLARES "this vantage is on the internet" — there is
-- no network_position field and no setup-wizard step: the intent is carried by
-- the act of provisioning, not by a stored enum (#124).
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
    host         TEXT NOT NULL,
    port         INTEGER NOT NULL DEFAULT 22 CHECK (port BETWEEN 1 AND 65535),
    username     TEXT NOT NULL,

    -- Availability is the one Derived property on this Declared term (CONTEXT.md):
    --   'pending'     — provisioned, no successful connection has pinned a host key yet
    --   'available'   — host key pinned, the position is reachable
    --   'unavailable' — a pinned host key later mismatched, or the position went unreachable
    availability TEXT NOT NULL DEFAULT 'pending'
                   CHECK (availability IN ('pending', 'available', 'unavailable')),

    -- The public half of the instance-generated SSH keypair, in authorized_keys
    -- form. NULL until the worker has generated the pair on its own volume. The
    -- private half never appears in this database.
    public_key   TEXT,

    -- The SSH host key pinned on first successful connection (trust-on-first-use),
    -- in known_hosts form. NULL until first connect.
    host_key     TEXT,

    created_by   BIGINT NOT NULL REFERENCES account (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A prober endpoint is provisioned once. Two vantages differing only in host key
-- or availability are the same declared position, so uniqueness is on the
-- (host, port, username) the operator dials.
CREATE UNIQUE INDEX vantage_endpoint_key ON vantage (host, port, username);

-- +goose Down
DROP TABLE vantage;
