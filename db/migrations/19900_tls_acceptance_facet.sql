-- +goose Up
-- Admit the `tls-acceptance` facet onto both the Observation and the Span timeline,
-- and ship its Scan (#199, ADR-0028). Unlike `certificate` (#197), which rides the
-- `reachability` exchange, `tls-acceptance` is an exchange — and a `Scan` — of its
-- OWN: a WEEKLY enumeration over every open `Service`, offering the full TLS
-- candidate set (version/cipher), with NO port list. Generalising the facet-agnostic
-- drift machinery to it is the same strictly additive act as every facet before it —
-- widen the two guards that name which facets may hold a timeline, and nothing else
-- moves (ADR-0011). `subject_kind` already admits `service` (18805/19000), the key
-- under which accepted versions/suites are single-valued, so only the facet set
-- widens here.
--
-- The CHECK is a closed union that grows one member at a time. This sets BOTH the
-- observation and span guards to the FULL v1 facet list — the six facets — which is
-- the sixth and final member, so the constraint reaches its settled v1 shape.
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity', 'tls-acceptance'));

ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity', 'tls-acceptance'));

-- The `tls-acceptance` Scan ships ENABLED at weekly cadence (604800 s). Weekly is a
-- cadence bought against N handshakes per Service (measurement-offers §1.5), never a
-- port tier: its scope is the open `Service` population plus the candidate set, read
-- at fan-out from the current `reachability` timelines, so with no reached Service
-- it produces no jobs — a legible empty state (CONTEXT.md `Scan`) — and needs no
-- opt-in machinery. It is a Declared Scan: a manual dispatch does not reset its
-- cadence, and its Batch records the candidate set it offered by content (ADR-0028).
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('tls-acceptance', TRUE, 604800);

-- +goose Down
DELETE FROM scan WHERE kind = 'tls-acceptance';

ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));

ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability', 'certificate', 'http-identity'));
