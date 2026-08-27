-- +goose Up
-- The `http-identity` Scan (v1 spec §3.3/§3.4, ADR-0011/ADR-0024) — the dispatch
-- half P0.11's first landing (child #686, commit f7cdb25) left inert. The prober
-- case, the probe User-Agent, the `http-exchange` measurer, the drift fold and the
-- four HTTP-identity rules all already shipped, but NOTHING emitted an
-- `http-exchange` job, so the `http-identity` facet was never persisted, every
-- Endpoint read `HTTPResponded = false`, and all four rules
-- (`plaintext-http-no-https`, `redirect-does-not-upgrade-to-tls`,
-- `redirect-to-host-outside-estate`, `unauthenticated-request-answered`) sat
-- permanently `outside-domain`. This Scan is that missing dispatch.
--
-- It rides the daily hot-Scan reachability the way `tls-acceptance` (#199) rides
-- the open-Service population: a Scan of its OWN, reading the current `reachability`
-- timelines reading `reached` at fan-out and issuing one `GET /` per reached
-- `Endpoint`, with NO port list — the Services carry their own ports (ADR-0028's
-- enumeration shape, ADR-0011's Endpoint keying). It dispatches the `http-exchange`
-- leaf, exactly as `hot`/`cold` dispatch `connect-outcome`: the Scan kind and the
-- leaf kind differ by design.
--
-- The kind CHECK is a closed union grown one member at a time (18801, widened for
-- `ct` in 21100). It is widened here rather than in 18801 so the kind set travels
-- with the dispatch that introduces it.
ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct', 'http-identity'));

-- Ships ENABLED at daily cadence (86400 s) — the reachability cadence it rides. A
-- single `GET /` per reached Endpoint is cheap (unlike the tls-acceptance
-- enumeration's N handshakes, which buys a weekly cadence), so it matches the daily
-- hot Scan. Its scope is the open `Service` population read at fan-out, so with no
-- reached Service — the shipped state before any hot Scan has run — it produces no
-- jobs: a legible empty state (CONTEXT.md `Scan`), needing no opt-in machinery.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('http-identity', TRUE, 86400);

-- +goose Down
DELETE FROM scan WHERE kind = 'http-identity';

ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct'));
