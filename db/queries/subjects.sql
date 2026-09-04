-- The gate carries the read instant, so no parameterless VIEW holds it and it inlines per read.

-- name: ListCurrentNameSubjects :many
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE value->>'outcome' NOT IN ('NameError', 'Shadowed')
  AND (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetNameSubject :one
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: GetNameCitation :one
-- A Citation never ages, so the admission hop reads under no live-tier clock (ADR-0096).
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
admission_hop AS (
    SELECT an.created_at AS observed_at, an.source, NULL::bigint AS vantage_id,
           an.batch_id, b.scan_id, sc.kind AS scan_kind, an.seed_id,
           'admission'::text AS hop_kind, 0 AS priority
    FROM admitted_name an
    JOIN batch b ON b.id = an.batch_id
    JOIN scan  sc ON sc.id = b.scan_id
    WHERE an.name = @subject_key
    ORDER BY an.id DESC
    LIMIT 1
),
observation_hop AS (
    SELECT o.observed_at, o.source, o.vantage_id,
           o.batch_id, b.scan_id, sc.kind AS scan_kind, NULL::bigint AS seed_id,
           'observation'::text AS hop_kind, 1 AS priority
    FROM live o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  sc ON sc.id = b.scan_id
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = @subject_key
    ORDER BY o.observed_at ASC, o.id ASC
    LIMIT 1
)
SELECT observed_at, source, vantage_id, batch_id, scan_id, scan_kind, seed_id, hop_kind
FROM (
    -- The leading arm's NULL seed_id is what makes sqlc type the column nullable.
    -- Swapping them to put the admission first is unnecessary: ORDER BY priority does that.
    SELECT observed_at, source, vantage_id, batch_id, scan_id, scan_kind, seed_id, hop_kind, priority
    FROM observation_hop
    UNION ALL
    SELECT observed_at, source, vantage_id, batch_id, scan_id, scan_kind, seed_id, hop_kind, priority
    FROM admission_hop
) h
ORDER BY priority
LIMIT 1;

-- name: ListCurrentServiceSubjects :many
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'service' AND o.facet = 'reachability'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetServiceSubject :one
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'service' AND o.facet = 'reachability' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: ListCurrentEndpointSubjects :many
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'endpoint' AND o.facet = 'http-identity'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetEndpointSubject :one
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
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'endpoint' AND o.facet = 'http-identity' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: FindNameCitingAddress :one
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
        o.subject_key AS subject_key,
        o.value->>'outcome' AS outcome,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.vantage_id, o.observed_at DESC
)
SELECT subject_key, observed_at
FROM latest
WHERE outcome = 'Resolved'
  AND value->'addresses' ? @address::text
ORDER BY observed_at ASC, subject_key ASC
LIMIT 1;

-- name: FindCoveringAddressSeed :one
SELECT s.id, s.address_cidr, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
WHERE s.kind = 'address' AND s.address_cidr IS NOT NULL
  AND s.address_cidr >>= @address::inet
ORDER BY masklen(s.address_cidr) DESC
LIMIT 1;

-- name: FindCoveringNameSeed :one
SELECT s.id, s.name_domain, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
WHERE s.kind = 'name' AND s.name_domain IS NOT NULL
  -- A declared domain is LDH-validated at declaration, so it carries no LIKE metacharacter.
  AND (@name::text = s.name_domain OR @name::text LIKE '%.' || s.name_domain)
ORDER BY length(s.name_domain) DESC
LIMIT 1;

-- name: FindNameSeedByID :one
SELECT s.id, s.name_domain, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
WHERE s.kind = 'name' AND s.id = @seed_id;
