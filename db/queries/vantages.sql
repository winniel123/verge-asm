-- name: CreateVantage :one
INSERT INTO vantage (name, resolver, host, port, username, availability, created_by)
VALUES (
    sqlc.arg(name)::text,
    sqlc.arg(resolver)::text,
    sqlc.arg(host)::text,
    sqlc.arg(port)::int,
    sqlc.arg(username)::text,
    'pending',
    sqlc.arg(created_by)::bigint
)
RETURNING id, name, class, resolver, host, port, username, availability,
          public_key, host_key, created_by, created_at, latency_ms, platform, egress,
          dialled_addr;

-- name: ListVantages :many
SELECT v.id, v.name, v.class, v.resolver, v.host, v.port, v.username,
       v.availability, v.public_key, v.host_key, v.created_by, v.created_at,
       v.latency_ms, v.platform, v.egress, v.dialled_addr,
       (EXISTS (SELECT 1 FROM observation o WHERE o.vantage_id = v.id)
        OR EXISTS (SELECT 1 FROM span sp WHERE sp.vantage_id = v.id))::boolean AS observed
FROM vantage v
WHERE v.host IS NOT NULL
ORDER BY v.created_at DESC, v.id DESC;

-- name: GetVantage :one
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE id = $1;

-- name: ListUnavailableVantages :many
SELECT id, name, class, resolver, availability
FROM vantage
WHERE availability = 'unavailable'
ORDER BY name;

-- name: ListAvailableProberVantageIDs :many
SELECT id
FROM vantage
WHERE availability = 'available' AND host IS NOT NULL
ORDER BY id;

-- name: ListVantagesNeedingKey :many
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE host IS NOT NULL AND public_key IS NULL
ORDER BY id;

-- name: ListVantagesNeedingLatency :many
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE host IS NOT NULL AND public_key IS NOT NULL AND latency_ms IS NULL
ORDER BY id;

-- name: SetVantageResolver :execrows
-- The resolver keys every timeline. Retention keeps span, not observation (ADR-0070, #1716).
UPDATE vantage
SET resolver = $2
WHERE vantage.id = $1
  AND NOT EXISTS (SELECT 1 FROM observation o WHERE o.vantage_id = $1)
  AND NOT EXISTS (SELECT 1 FROM span sp WHERE sp.vantage_id = $1);

-- name: SetVantageLatency :exec
UPDATE vantage
SET latency_ms = $2
WHERE id = $1;

-- name: SetVantageProbeFacts :exec
UPDATE vantage
SET platform = $2, egress = $3, dialled_addr = $4
WHERE id = $1;

-- name: SetVantagePublicKey :exec
UPDATE vantage
SET public_key = $2
WHERE id = $1;

-- name: PinVantageHostKey :exec
UPDATE vantage
SET host_key = $2,
    -- Availability is concluded from a batch outcome, and a connect is not one (ADR-0108).
    availability = CASE WHEN availability = 'unavailable' THEN availability ELSE 'available' END
WHERE id = $1 AND host_key IS NULL;

-- name: MarkVantageUnavailable :exec
-- A vantage that becomes unavailable closes its open spans here, so no composition read
-- carries an availability predicate.
WITH became_unavailable AS (
    -- NULL has never been unavailable, so a resolver-only vantage still closes (ADR-2087).
    UPDATE vantage
    SET availability = 'unavailable'
    WHERE vantage.id = $1
    RETURNING vantage.id AS vantage_id
),
closed AS (
    -- Every facet, named nowhere: one added later needs no query edit (ADR-2087, #2144).
    UPDATE span
    SET closed_at = now()
    WHERE span.closed_at IS NULL
      AND span.vantage_id IN (SELECT became_unavailable.vantage_id FROM became_unavailable)
      -- The guard is on the spans, so a second mark reaches what opened since (#2060).
      AND (span.is_gap AND span.value ->> 'cause' = 'vantage-unavailable') IS NOT TRUE
    RETURNING span.subject_kind, span.subject_key, span.facet, span.discriminator,
              span.vantage_id, span.source, span.derivation
)
INSERT INTO span (
    subject_kind, subject_key, facet, discriminator, vantage_id, source,
    value, is_gap, derivation, opened_at
)
SELECT subject_kind, subject_key, facet, discriminator, vantage_id, source,
       -- A spelling, never a scope: reachability alone decodes a lowercase outcome
       -- (connectoutcome.GapOutcome) and every other facet the capitalised one, so a facet
       -- added later takes the default rather than a new branch.
       CASE facet
           WHEN 'reachability' THEN '{"outcome":"gap","cause":"vantage-unavailable","reason":"we could not look from this position"}'::jsonb
           ELSE '{"outcome":"Gap","cause":"vantage-unavailable"}'::jsonb
       END,
       TRUE,
       -- No leaf ran, so the Gap carries the closed span's vector (ADR-0014).
       derivation,
       now()
FROM closed;

-- name: MarkVantageAvailable :exec
-- Recovery retires the outage Gap on the facets the recovering batch re-read (ADR-2087).
WITH became_available AS (
    UPDATE vantage
    SET availability = 'available'
    WHERE vantage.id = sqlc.arg(id)
    RETURNING vantage.id AS vantage_id
)
UPDATE span
SET closed_at = now()
WHERE span.closed_at IS NULL
  AND span.vantage_id IN (SELECT became_available.vantage_id FROM became_available)
  -- A resolver signal measures no port, so it retires no connect Gap (ADR-2087, #2060).
  AND span.facet = ANY(sqlc.arg(facets)::text[])
  -- A connect batch keeps opening reached spans behind an outage (#2060).
  AND span.is_gap
  AND span.value ->> 'cause' = 'vantage-unavailable';
