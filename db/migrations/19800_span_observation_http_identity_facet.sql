-- +goose Up
-- Admit the `http-identity` facet onto the Observation and Span timelines (#198).
-- The drift engine is facet-agnostic — the fold, Span, Break, Gap and Transition
-- do not vary by facet (ADR-0011) — so generalising it to a new facet is strictly
-- additive: widen the two guards that name which facets may hold a timeline, and
-- nothing else moves. `http-identity` on an `Endpoint` subject (the `(Name,
-- Service)` pair, the only key under which HTTP identity is single-valued) is the
-- fourth facet the machinery serves, after `resolution`/`dns-record` on `Name` and
-- `reachability` on `Service`. The subject_kind CHECK already admits 'endpoint'
-- (18805/19000), so only the facet set widens here.
--
-- CONVERGENCE (wave-4): #197 (certificate) widens these same two CHECKs
-- concurrently. Both migrations therefore set the constraint to the FULL v1 facet
-- list — `certificate` is included here even though this ticket does not produce
-- it — so that whichever of the two applies second lands on the identical final
-- constraint rather than a partial union that depends on apply order. The CHECK is
-- a closed union that the two tickets converge on, never a hardcode either owns.
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));

ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));

-- +goose Down
ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));

ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));
