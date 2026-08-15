-- +goose Up
-- A Scan is the operator's configured recurring intent — the configured thing,
-- never the executed one (CONTEXT.md `Scan`). v1 has five, two of them port
-- tiers; this ticket ships the `dns` Scan (v1 spec §3.4, ADR-0084): daily,
-- unconditional of Custody, no port list, every configured Vantage, covering
-- `resolution` and our own resolver's `dns-record`. A cadence is not aperture,
-- so it lives here rather than on a Batch's recorded scope.
CREATE TABLE scan (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind            TEXT NOT NULL UNIQUE CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns')),
    enabled         BOOLEAN NOT NULL,
    cadence_seconds BIGINT NOT NULL CHECK (cadence_seconds > 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The dns Scan ships enabled at daily — the tightest shipped port tier's
-- cadence (ADR-0084). Loosening it withdraws Addresses rather than probing
-- stale ones, which needs no coupling machinery.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('dns', TRUE, 86400);

-- +goose Down
DROP TABLE scan;
