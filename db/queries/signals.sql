-- Reads behind the Signals screen (#202). All three observation reads are additive
-- read queries over the wave-0/1 corpus (observation / vantage) — no new schema.
-- The web layer folds these into the per-Name Derived snapshot the `Signal`
-- engine (internal/signal) evaluates: the five Name-only rules read `resolution`
-- (which `resolution-walk` and `wildcard-discrimination` decide jointly), the
-- `dns-record` CNAME target and NS delegation, and the operator's zone file.
--
-- Every observation read here reads through the live-tier gate (#237, ADR-0041):
-- the `cover`/`live` CTE pair is the inlined twin of
-- ListLiveObservationsForDerivation (db/queries/retention.sql), evaluated against
-- the caller's read instant @as_of with k = @floor_cadences, so the Signal engine
-- never folds an evidential row into a Derived snapshot. ListZoneDeclarations
-- reads the operator's supplied zone file (input, not measured) and is not gated.

-- name: ListNameResolutionsByClass :many
-- The latest `resolution` observation per (Name, Vantage class). The engine folds
-- these into a cross-class composed outcome (for the four cross-class rules) and
-- keeps the internet-class view apart (for the one vantage-scoped rule, ADR-0071).
-- DISTINCT ON keeps the most recent value per (name, class). Reads through the
-- live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key, v.class)
        o.subject_key AS subject_key,
        v.class       AS class,
        o.value       AS value
    FROM live o
    JOIN vantage v ON v.id = o.vantage_id
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, v.class, o.observed_at DESC, o.id DESC
)
SELECT subject_key, class, value
FROM latest
ORDER BY subject_key, class;

-- name: ListNameDNSRecords :many
-- The latest `dns-record` observation per (Name, qtype discriminator). The engine
-- reads two of these: the CNAME discriminator carries the alias target (for
-- cname-target-name-error) and the NS discriminator carries the delegation walk's
-- Lame verdict (folded into the composed resolution the rules read). Reads through
-- the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key, o.discriminator)
        o.subject_key   AS subject_key,
        o.discriminator AS discriminator,
        o.value         AS value
    FROM live o
    WHERE o.facet = 'dns-record' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.discriminator, o.observed_at DESC, o.id DESC
)
SELECT subject_key, discriminator, value
FROM latest
ORDER BY subject_key, discriminator;

-- name: ListServiceReachabilitySpansByClass :many
-- The CURRENT `reachability` span per (Service, Vantage class) (#254, ADR-0104).
-- buildServiceFacts reads the SPAN, not the latest observation, because the span
-- carries `is_gap`: a blanket responder's reach is a Gap, and a Gap leg reads as
-- absent (HasInternetReach=false) so `sensitive-port-reached-from-internet`
-- returns not-evaluable with no rule edit, and a Gap is not a `reachability` value
-- so a blanket responder's ports drop out of any open-port count without a special
-- case. Span reads are NOT routed through the live-tier observation gate (#237):
-- the span corpus is the already-derived timeline the fold produced, kept forever
-- (ADR-0041), so an as_of bound would wrongly hide settled state rather than
-- protect a re-derivation. DISTINCT ON keeps the most recent OPEN span per
-- (service, class), mirroring the observation read one facet over.
SELECT DISTINCT ON (sp.subject_key, v.class)
    sp.subject_key AS subject_key,
    v.class        AS class,
    sp.value       AS value,
    sp.is_gap      AS is_gap
FROM span sp
JOIN vantage v ON v.id = sp.vantage_id
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
ORDER BY sp.subject_key, v.class, sp.opened_at DESC, sp.id DESC;

-- name: ListBlanketedReachServices :many
-- Every Service whose CURRENT `reachability` span is a Gap — a blanket responder,
-- or an address whose control probe could not complete (ADR-0104). The Coverage
-- aperture register (#254, ADR-0095) reads this to state, in prose, that these
-- addresses answer on all ports and are a proxy edge rather than the origin — a
-- read surface, never a Transition or a new message cause. The caller folds the
-- Service keys to their distinct Addresses; a Gap span is never routed through the
-- live-tier gate for the reason above.
SELECT DISTINCT sp.subject_key AS subject_key
FROM span sp
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
  AND sp.is_gap = TRUE
ORDER BY sp.subject_key;

-- name: ListEndpointCertificates :many
-- The latest `certificate` observation per Endpoint (#203) — the value the six
-- certificate rules and `plaintext-http-no-https` read. The value is the closed
-- union `presented(chain) | tls-refused | no-tls`; the engine reads the outcome
-- tag. The parsed leaf attributes the five certificate-detail rules need are not
-- stored (only the fingerprint chain is), so those rules render a presented chain
-- `not-evaluable` until a certificate-parsing leaf lands. DISTINCT ON keeps the
-- most recent value per Endpoint. Reads through the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value
    FROM live o
    WHERE o.subject_kind = 'endpoint' AND o.facet = 'certificate'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value
FROM latest
ORDER BY subject_key;

-- name: ListZoneDeclarations :many
-- The latest supplied zone file per name-scope Seed, with its declared domain and
-- content, so the web layer can extract the owner names the operator declares
-- (signal.DeclaredNames) — the domain of the two zone rules. One row per Seed;
-- DISTINCT ON keeps the most recent supply. This reads the operator's supplied
-- zone file — input, not a measurement — so it does not pass the live-tier gate.
SELECT DISTINCT ON (z.seed_id)
    s.name_domain AS name_domain,
    z.content     AS content
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
WHERE s.kind = 'name' AND s.name_domain IS NOT NULL
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;
