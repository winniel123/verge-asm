-- +goose Up
-- The Postgres-backed queue (v1 spec §4.1): dispatched with
-- `SELECT … FOR UPDATE SKIP LOCKED` + `LISTEN/NOTIFY`, not a broker. One job is
-- one Batch. A job holds the attempted scope and the offers to put on the wire;
-- its Batch is created only at a terminal outcome (completed or dead-lettered),
-- in the same transaction that writes the outcome and observations.
--
-- Retry is always a **new** Batch, never a resumption: a transient failure that
-- has attempts left enqueues a fresh job (a new row) rather than re-opening a
-- partial one, so there is no resumption path to resume.
CREATE TABLE queue_job (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    scan_id         BIGINT NOT NULL REFERENCES scan (id),
    vantage_id      BIGINT REFERENCES vantage (id),
    dispatch_id     BIGINT REFERENCES dispatch (id),
    kind            TEXT NOT NULL,
    spec            JSONB NOT NULL,
    attempted_scope JSONB NOT NULL,
    offers          JSONB NOT NULL,
    state           TEXT NOT NULL DEFAULT 'ready'
                    CHECK (state IN ('ready', 'running', 'done', 'dead', 'retried')),
    attempt         INT NOT NULL DEFAULT 1,
    max_attempts    INT NOT NULL DEFAULT 5,
    batch_id        BIGINT REFERENCES batch (id),
    run_after       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The dispatch index the claim query rides: ready jobs whose run_after has
-- passed, oldest first. FOR UPDATE SKIP LOCKED reads it under concurrency.
CREATE INDEX queue_job_ready_idx ON queue_job (run_after) WHERE state = 'ready';

-- +goose Down
DROP TABLE queue_job;
