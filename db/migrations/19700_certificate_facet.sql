-- +goose Up
-- Admit the `certificate` facet onto both the Observation and the Span timeline
-- (#197). Like `reachability` before it (#195), the certificate value rides an
-- existing exchange rather than a Scan of its own: its TLS handshake is a step
-- inside the `reachability` exchange, so generalising the drift machinery to it is
-- the same strictly additive act — widen the guard that names which facets may
-- hold a timeline and nothing else moves (ADR-0011, ADR-0028). The tables' shape,
-- indexes and retention bar are untouched, and `subject_kind` already admits
-- `endpoint`, the key under which a presented chain is single-valued.
--
-- The CHECK is a closed union that grows one member at a time. To converge with
-- #198 (http-identity), which widens these same two guards concurrently, this
-- migration sets BOTH to the FULL v1 facet list — `http-identity` included even
-- though #197 does not produce it — so the two tickets' migrations reach the same
-- final constraint regardless of apply order.
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));

ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));

-- +goose Down
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));

ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));
