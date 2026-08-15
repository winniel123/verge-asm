-- +goose Up
-- Admit the `reachability` facet onto the Span timeline (#195). The drift engine
-- is facet-agnostic — the fold, Span, Break, Gap and Transition do not vary by
-- facet (ADR-0011) — so generalising it to a new facet is a strictly additive
-- act: widen the guard that names which facets may hold a timeline, and nothing
-- else moves. The Span table's shape, indexes and retention bar are untouched.
--
-- `reachability` on a `Service` subject is the second facet the machinery serves
-- (after `resolution`/`dns-record` on `Name`); the wave-4 facet tickets (#197
-- certificate, #198 http-identity) each widen this same guard for their own
-- facet, so the CHECK is a closed union that grows one member at a time rather
-- than a hardcode any one ticket owns.
ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));

-- +goose Down
ALTER TABLE span DROP CONSTRAINT span_facet_check;
ALTER TABLE span ADD CONSTRAINT span_facet_check
    CHECK (facet IN ('resolution', 'dns-record'));
