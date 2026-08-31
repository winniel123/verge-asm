-- +goose Up
-- The measured reliability bar for the bulk CT primary (spec
-- docs/spec/ct-source-replacement.md §3, map #854, ticket #879). Each bulk-by-name
-- query records one sample here: whether it succeeded, its end-to-end fetch latency,
-- and whether a successful query returned zero certificate names (the false-empty
-- limb). The bar is measured over a rolling window of the newest samples per source,
-- never asserted once — the worker trims each source to the window size on write, and
-- the web reads the window to report pass/fail per limb. crt.sh is exempt as the
-- keyless fallback (spec §3); its samples still accrue for contrast.
CREATE TABLE ct_reliability_sample (
    id          BIGSERIAL PRIMARY KEY,
    source      TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ok          BOOLEAN NOT NULL,
    latency_ms  BIGINT NOT NULL,
    empty       BOOLEAN NOT NULL
);

-- The read window and the write-time trim both order newest-first per source; id
-- breaks ties for samples that share an observed_at instant.
CREATE INDEX ct_reliability_sample_source_time
    ON ct_reliability_sample (source, observed_at DESC, id DESC);

-- +goose Down
DROP TABLE ct_reliability_sample;
