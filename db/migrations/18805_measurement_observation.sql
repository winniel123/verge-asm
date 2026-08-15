-- +goose Up
-- An Observation is a single measured fact: at a time, from a vantage, in a
-- batch, a source reported that a subject had a given value for a given facet
-- (CONTEXT.md `Observation`). One concept across every facet, so change
-- detection is written once. The timeline key is
-- (subject, facet, discriminator, vantage, source); `discriminator` carries the
-- qtype for `dns-record` and is empty for `resolution`. `observed_at` is when
-- the source spoke — for our own resolver that is our read.
CREATE TABLE observation (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id      BIGINT NOT NULL REFERENCES batch (id),
    facet         TEXT NOT NULL CHECK (facet IN ('resolution', 'dns-record')),
    subject_kind  TEXT NOT NULL CHECK (subject_kind IN ('name', 'address', 'service', 'endpoint')),
    subject_key   TEXT NOT NULL,
    discriminator TEXT NOT NULL DEFAULT '',
    vantage_id    BIGINT REFERENCES vantage (id),
    source        TEXT NOT NULL DEFAULT 'resolver',
    value         JSONB NOT NULL,
    observed_at   TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX observation_timeline_idx
    ON observation (subject_key, facet, discriminator, vantage_id, source, observed_at DESC);
CREATE INDEX observation_batch_idx ON observation (batch_id);

-- +goose Down
DROP TABLE observation;
