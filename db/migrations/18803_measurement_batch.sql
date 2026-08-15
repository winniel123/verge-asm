-- +goose Up
-- A Batch is one queue job's outcome (v1 spec §4.1): one queue job is one
-- Batch. Job outcome and observation data commit together, so a Batch row is
-- written in the same transaction as the Observations it produced. It records
-- the offers and the completed scope **by content** (ADR-0025) — never a
-- library default and never the attempted scope. A dead-lettered Batch records
-- an **empty** scope, never its attempted one, because asserting the attempted
-- scope would manufacture absences it never measured.
CREATE TABLE batch (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scan_id        BIGINT NOT NULL REFERENCES scan (id),
    dispatch_id    BIGINT REFERENCES dispatch (id),
    vantage_id     BIGINT REFERENCES vantage (id),
    kind           TEXT NOT NULL,
    outcome        TEXT NOT NULL CHECK (outcome IN ('completed', 'dead-lettered')),
    offers         JSONB NOT NULL,
    recorded_scope JSONB NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX batch_scan_created_idx ON batch (scan_id, created_at DESC);

-- +goose Down
DROP TABLE batch;
