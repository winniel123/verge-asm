-- Reads behind the Signals screen (#202). All three are additive read queries
-- over the wave-0/1 corpus (observation / vantage / zone_file) — no new schema.
-- The web layer folds these into the per-Name Derived snapshot the `Signal`
-- engine (internal/signal) evaluates: the five Name-only rules read `resolution`
-- (which `resolution-walk` and `wildcard-discrimination` decide jointly), the
-- `dns-record` CNAME target and NS delegation, and the operator's zone file.

-- name: ListNameResolutionsByClass :many
-- The latest `resolution` observation per (Name, Vantage class). The engine folds
-- these into a cross-class composed outcome (for the four cross-class rules) and
-- keeps the internet-class view apart (for the one vantage-scoped rule, ADR-0071).
-- DISTINCT ON keeps the most recent value per (name, class).
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key, v.class)
        o.subject_key AS subject_key,
        v.class       AS class,
        o.value       AS value
    FROM observation o
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
-- Lame verdict (folded into the composed resolution the rules read).
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key, o.discriminator)
        o.subject_key   AS subject_key,
        o.discriminator AS discriminator,
        o.value         AS value
    FROM observation o
    WHERE o.facet = 'dns-record' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.discriminator, o.observed_at DESC, o.id DESC
)
SELECT subject_key, discriminator, value
FROM latest
ORDER BY subject_key, discriminator;

-- name: ListServiceReachabilityByClass :many
-- The latest `reachability` observation per (Service, Vantage class) (#203). The
-- engine reads the internet-class leg for `sensitive-port-reached-from-internet`
-- (ADR-0071: a class-scoped internet, existential composition — the internal twin
-- is a different, refused rule). DISTINCT ON keeps the most recent value per
-- (service, class), mirroring the Name resolution read one facet over.
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key, v.class)
        o.subject_key AS subject_key,
        v.class       AS class,
        o.value       AS value
    FROM observation o
    JOIN vantage v ON v.id = o.vantage_id
    WHERE o.facet = 'reachability' AND o.subject_kind = 'service'
    ORDER BY o.subject_key, v.class, o.observed_at DESC, o.id DESC
)
SELECT subject_key, class, value
FROM latest
ORDER BY subject_key, class;

-- name: ListEndpointCertificates :many
-- The latest `certificate` observation per Endpoint (#203) — the value the six
-- certificate rules and `plaintext-http-no-https` read. The value is the closed
-- union `presented(chain) | tls-refused | no-tls`; the engine reads the outcome
-- tag. The parsed leaf attributes the five certificate-detail rules need are not
-- stored (only the fingerprint chain is), so those rules render a presented chain
-- `not-evaluable` until a certificate-parsing leaf lands. DISTINCT ON keeps the
-- most recent value per Endpoint.
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value
    FROM observation o
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
-- DISTINCT ON keeps the most recent supply.
SELECT DISTINCT ON (z.seed_id)
    s.name_domain AS name_domain,
    z.content     AS content
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
WHERE s.kind = 'name' AND s.name_domain IS NOT NULL
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;
