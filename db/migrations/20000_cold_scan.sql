-- +goose Up
-- The cold Scan (v1 spec §3.4, ADR-0044): the full 1–65535 TCP port tier,
-- monthly, opt-in **per `Seed` scope**. It ships configured and **DISABLED**
-- with an empty scope list, and never runs unasked — not at onboarding, not on
-- config save. A one-off full-range sweep has no cadence and therefore no
-- currency bound, so it is not an expressible object; the only expressible form
-- of "unasked" is shipping this Scan enabled at monthly cadence, and that is the
-- option ADR-0044 refused. It exists disabled so that "the operator has not
-- enabled the full-range tier" is a legible state rather than an absence. The
-- tier is enabled by opting a `Seed` scope in (cold_scan_scope below), which
-- flips this row's enabled flag; it then fans out only on this monthly cadence.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('cold', FALSE, 2592000);

-- The per-`Seed` opt-in: one row is one Seed scope opted into the cold tier.
-- Enabling is per-Seed, not global (ADR-0044) — the operator names which scopes
-- the full-range sweep covers, and the cold Scan probes only Custody-admitted
-- addresses that an opted-in scope enumerates. Opting a scope in is an ordinary
-- aperture widening; opting the last one out returns the tier to its disabled,
-- empty-scope shipped state. The unique index makes a re-opt-in idempotent.
CREATE TABLE cold_scan_scope (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seed_id    BIGINT NOT NULL REFERENCES seed (id) ON DELETE CASCADE,
    created_by BIGINT NOT NULL REFERENCES account (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX cold_scan_scope_seed_key ON cold_scan_scope (seed_id);

-- +goose Down
DROP TABLE cold_scan_scope;
DELETE FROM scan WHERE kind = 'cold';
