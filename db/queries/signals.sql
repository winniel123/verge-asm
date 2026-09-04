-- name: ListNameResolutionsByClass :many
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
    SELECT DISTINCT ON (o.subject_key, o.vantage_id)
        o.subject_key  AS subject_key,
        o.vantage_id   AS vantage_id,
        o.value        AS value,
        o.observed_at  AS observed_at,
        o.id           AS id,
        v.host         AS host,
        v.egress       AS egress,
        v.dialled_addr AS dialled_addr
    FROM live o
    JOIN vantage v ON v.id = o.vantage_id
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.vantage_id, o.observed_at DESC, o.id DESC
)
SELECT subject_key, vantage_id, value, observed_at, id, host, egress, dialled_addr
FROM latest
ORDER BY subject_key, vantage_id;

-- name: ListNameDNSRecords :many
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
-- The span corpus is already derived, so an as_of bound would hide settled state (ADR-0105).
SELECT DISTINCT ON (sp.subject_key, sp.vantage_id)
    sp.subject_key AS subject_key,
    sp.vantage_id  AS vantage_id,
    sp.value       AS value,
    sp.is_gap      AS is_gap,
    sp.opened_at   AS opened_at,
    sp.id          AS id,
    v.host         AS host,
    v.egress       AS egress,
    v.dialled_addr AS dialled_addr
FROM span sp
JOIN vantage v ON v.id = sp.vantage_id
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
ORDER BY sp.subject_key, sp.vantage_id, sp.opened_at DESC, sp.id DESC;

-- name: ListServiceTLSAcceptance :many
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
    WHERE o.subject_kind = 'service' AND o.facet = 'tls-acceptance'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value
FROM latest
ORDER BY subject_key;

-- name: ListBlanketedReachServices :many
SELECT DISTINCT sp.subject_key AS subject_key
FROM span sp
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
  AND sp.is_gap = TRUE
ORDER BY sp.subject_key;

-- name: ListEndpointCertificates :many
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
SELECT DISTINCT ON (z.seed_id)
    s.name_domain AS name_domain,
    z.content     AS content
FROM zone_file z
JOIN seed s ON s.id = z.seed_id
WHERE s.kind = 'name' AND s.name_domain IS NOT NULL
ORDER BY z.seed_id, z.supplied_at DESC, z.id DESC;

-- name: MintSignalInstances :exec
INSERT INTO signal_instance (signal_name, subject_key)
SELECT unnest(sqlc.arg(signal_names)::text[]), unnest(sqlc.arg(subject_keys)::text[])
ON CONFLICT (signal_name, subject_key) DO NOTHING;

-- name: ListSignalInstances :many
SELECT id, signal_name, subject_key, first_seen
FROM signal_instance
ORDER BY signal_name, subject_key;
