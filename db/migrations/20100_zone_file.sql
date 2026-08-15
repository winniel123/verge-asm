-- +goose Up
-- The operator's own zone file is Tier-0 ground truth (v1 spec §3.1): declared
-- authority, uploaded — never mounted, because a supply act needs an instant —
-- through `web`, and stored here in the shared database so both `web` and
-- `worker` can read it. It is **not a secret** (§4.2/§4.3): it is evidence, held
-- where every service that reads it can, unlike the session key or prober key
-- which live on a single service's own volume.
--
-- One row is one supply act for one name-scope Seed. A re-export is a **new**
-- row with a new `supplied_at`, never an update: the zone Scan restates the
-- latest supplied file (CONTEXT.md `Observation` — a re-supplied file with
-- different contents is a new supply, and its instant is the bound's origin).
CREATE TABLE zone_file (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seed_id     BIGINT NOT NULL REFERENCES seed (id),
    supplied_at TIMESTAMPTZ NOT NULL,
    content     TEXT NOT NULL,
    uploaded_by BIGINT NOT NULL REFERENCES account (id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The dispatch read is "the latest supplied file per Seed": a covering index on
-- (seed_id, supplied_at DESC) serves the DISTINCT ON without a sort.
CREATE INDEX zone_file_latest_idx ON zone_file (seed_id, supplied_at DESC, id DESC);

-- The zone Scan (v1 spec §3.4): worker-read, no port list, no vantage choice.
-- Its cadence is the operator's declared re-supply interval — their promise
-- about how often they will re-export — shipped at monthly (30 days). It ships
-- enabled: with no zone file supplied its scope is empty, which is a legible
-- state producing no jobs (CONTEXT.md `Scan`), so it needs no coupling to the
-- first upload to be safe. The interval is operator-configurable via this row's
-- cadence_seconds.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('zone', TRUE, 2592000);

-- +goose Down
DELETE FROM scan WHERE kind = 'zone';
DROP TABLE zone_file;
