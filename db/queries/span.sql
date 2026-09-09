-- An as-of bound here would hide settled history rather than protect a derivation (ADR-0105).

-- name: GetOpenSpan :one
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE closed_at IS NULL
  AND subject_key = @subject_key
  AND facet = @facet
  AND discriminator = @discriminator
  AND vantage_id IS NOT DISTINCT FROM sqlc.narg('vantage_id')::bigint
  AND source = @source;

-- name: OpenSpan :one
INSERT INTO span (
    subject_kind, subject_key, facet, discriminator, vantage_id, source,
    value, is_gap, derivation, opened_at, opened_batch_id, opened_aperture
) VALUES (
    @subject_kind, @subject_key, @facet, @discriminator, sqlc.narg('vantage_id')::bigint,
    @source, @value, @is_gap, @derivation, @opened_at, sqlc.narg('opened_batch_id')::bigint,
    @opened_aperture
)
RETURNING id;

-- name: CloseSpan :exec
UPDATE span
SET closed_at = @closed_at,
    closure_reason = sqlc.narg('closure_reason'),
    closed_batch_id = sqlc.narg('closed_batch_id')::bigint
WHERE id = @id AND closed_at IS NULL;

-- name: ListOpenSpansForSubject :many
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE subject_kind = @subject_kind AND subject_key = @subject_key AND closed_at IS NULL
ORDER BY facet, discriminator, vantage_id, source;

-- name: ListAllOpenSpans :many
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE closed_at IS NULL
ORDER BY subject_kind, subject_key, facet, discriminator, vantage_id, source;

-- name: ListOpenEndpointCertificateSpans :many
SELECT s.subject_key, s.vantage_id, s.value, o.observed_at
FROM span s
JOIN LATERAL (
    -- The clock class reads the latest observation's age, not the span's (ADR-0043).
    SELECT observed_at FROM observation o
    WHERE o.subject_key = s.subject_key AND o.facet = s.facet AND o.discriminator = s.discriminator
      AND o.vantage_id IS NOT DISTINCT FROM s.vantage_id AND o.source = s.source
    ORDER BY o.observed_at DESC, o.id DESC
    LIMIT 1
) o ON TRUE
WHERE s.subject_kind = 'endpoint' AND s.facet = 'certificate'
  AND s.closed_at IS NULL AND NOT s.is_gap
ORDER BY s.subject_key, s.vantage_id, s.id;

-- name: ListSpansOpenSince :many
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE closed_at IS NULL OR closed_at > @since
ORDER BY subject_kind, subject_key, facet, discriminator, vantage_id, source, opened_at;

-- name: ListServiceReachabilitySpansByClassAt :many
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
  AND sp.opened_at <= @at
  AND (sp.closed_at IS NULL OR sp.closed_at > @at)
ORDER BY sp.subject_key, sp.vantage_id, sp.opened_at DESC, sp.id DESC;

-- name: ListServiceReachabilitySpansByClassAtForServices :many
-- The bound limits the per-job read to the batch's Services, not the corpus (ADR-0226 §1, #1609).
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
  AND sp.subject_key = ANY(sqlc.arg(service_keys)::text[])
  AND sp.opened_at <= @at
  AND (sp.closed_at IS NULL OR sp.closed_at > @at)
ORDER BY sp.subject_key, sp.vantage_id, sp.opened_at DESC, sp.id DESC;

-- name: ListSpansForSubject :many
SELECT s.id, s.subject_kind, s.subject_key, s.facet, s.discriminator, s.vantage_id, s.source,
       s.value, s.is_gap, s.derivation, s.opened_at, s.closed_at, s.closure_reason,
       v.name AS vantage_name
FROM span s
LEFT JOIN vantage v ON v.id = s.vantage_id
WHERE s.subject_kind = @subject_kind AND s.subject_key = @subject_key
ORDER BY s.facet, s.discriminator, s.vantage_id, s.source, s.opened_at, s.id;

-- name: ListRecentDriftEvents :many
SELECT
    'opened'::text   AS role,
    b.id             AS batch_id,
    b.kind           AS batch_kind,
    b.created_at     AS batch_at,
    b.recorded_scope AS recorded_scope,
    sp.subject_kind, sp.subject_key, sp.facet, sp.discriminator,
    sp.value, sp.is_gap, sp.derivation,
    sp.opened_at, sp.closed_at, sp.closure_reason,
    sp.opened_aperture  AS opened_aperture,
    pred.value          AS prev_value,
    pred.derivation     AS prev_derivation,
    pred.closed_at      AS prev_closed_at,
    pred.closure_reason AS prev_closure_reason,
    -- A Break on any resolution witness open at this instant voids returned (ADR-0097).
    EXISTS (
        SELECT 1
        FROM span w
        WHERE w.subject_kind = sp.subject_kind
          AND w.subject_key = sp.subject_key
          AND w.facet = 'resolution'
          AND w.opened_at <= sp.opened_at
          AND (w.closed_at IS NULL OR w.closed_at > sp.opened_at)
          AND w.derivation <> (
              SELECT wp.derivation
              FROM span wp
              WHERE wp.subject_kind = w.subject_kind
                AND wp.subject_key = w.subject_key
                AND wp.facet = w.facet
                AND wp.discriminator = w.discriminator
                AND wp.vantage_id IS NOT DISTINCT FROM w.vantage_id
                AND wp.source = w.source
                AND (wp.opened_at < w.opened_at OR (wp.opened_at = w.opened_at AND wp.id < w.id))
              ORDER BY wp.opened_at DESC, wp.id DESC
              LIMIT 1
          )
    )::boolean AS witness_broke
FROM span sp
JOIN batch b ON b.id = sp.opened_batch_id
LEFT JOIN LATERAL (
    SELECT p.value, p.derivation, p.closed_at, p.closure_reason
    FROM span p
    WHERE p.subject_kind = sp.subject_kind
      AND p.subject_key = sp.subject_key
      AND p.facet = sp.facet
      AND p.discriminator = sp.discriminator
      AND p.vantage_id IS NOT DISTINCT FROM sp.vantage_id
      AND p.source = sp.source
      AND (p.opened_at < sp.opened_at OR (p.opened_at = sp.opened_at AND p.id < sp.id))
    ORDER BY p.opened_at DESC, p.id DESC
    LIMIT 1
) pred ON true
WHERE b.created_at >= @since
  AND (sqlc.narg('until')::timestamptz IS NULL OR b.created_at < sqlc.narg('until')::timestamptz)

UNION ALL

SELECT
    'closed'::text   AS role,
    b.id             AS batch_id,
    b.kind           AS batch_kind,
    b.created_at     AS batch_at,
    b.recorded_scope AS recorded_scope,
    sp.subject_kind, sp.subject_key, sp.facet, sp.discriminator,
    sp.value, sp.is_gap, sp.derivation,
    sp.opened_at, sp.closed_at, sp.closure_reason,
    FALSE              AS opened_aperture,
    NULL::jsonb        AS prev_value,
    NULL::jsonb        AS prev_derivation,
    NULL::timestamptz  AS prev_closed_at,
    NULL::text         AS prev_closure_reason,
    FALSE              AS witness_broke
FROM span sp
JOIN batch b ON b.id = sp.closed_batch_id
WHERE b.created_at >= @since
  AND (sqlc.narg('until')::timestamptz IS NULL OR b.created_at < sqlc.narg('until')::timestamptz)
  -- A value-move close rides its successor's opened row, so counting it doubles the transition.
  AND sp.closure_reason IS NOT NULL

ORDER BY batch_at DESC, batch_id DESC, subject_kind, subject_key, facet, discriminator, opened_at
LIMIT @max_events;

-- name: ListWithdrawalLifespans :many
SELECT DISTINCT ON (w.subject_kind, w.subject_key, w.closed_at)
    w.subject_kind AS subject_kind,
    w.subject_key  AS subject_key,
    w.closed_at    AS withdrawn_at,
    fa.first_opened AS first_opened
FROM span w
JOIN LATERAL (
    SELECT MIN(p.opened_at)::timestamptz AS first_opened
    FROM span p
    WHERE p.subject_kind = w.subject_kind
      AND p.subject_key = w.subject_key
) fa ON TRUE
WHERE w.closure_reason IS NOT NULL
  AND w.closed_at IS NOT NULL
  AND w.closed_at >= @since
ORDER BY w.subject_kind, w.subject_key, w.closed_at, w.id;

-- name: ListSubjectFirstAppearances :many
SELECT
    sp.subject_kind AS subject_kind,
    sp.subject_key  AS subject_key,
    MIN(sp.opened_at)::timestamptz AS first_opened
FROM span sp
WHERE sp.subject_kind IN ('name', 'service')
GROUP BY sp.subject_kind, sp.subject_key
HAVING MIN(sp.opened_at) >= @since
ORDER BY first_opened, sp.subject_kind, sp.subject_key;

-- name: ListReachedServices :many
SELECT sp.subject_key AS service_key, sp.vantage_id AS vantage_id
FROM span sp
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
  AND sp.is_gap = FALSE
  AND (sp.value ->> 'outcome') = 'reached'
ORDER BY sp.vantage_id, sp.subject_key;

-- name: ListCitedAddressSpansForNames :many
-- The candidate set is what the departed Names ever cited, never every Address (ADR-0198 §1).
WITH cited AS (
    SELECT DISTINCT a.addr
    FROM span n
    CROSS JOIN LATERAL jsonb_array_elements_text(
        CASE WHEN jsonb_typeof(n.value -> 'addresses') = 'array' THEN n.value -> 'addresses' END
    ) AS a(addr)
    WHERE n.subject_kind = 'name'
      AND n.facet = 'resolution'
      AND n.subject_key = ANY(sqlc.arg(names)::text[])
)
SELECT s.id, s.subject_key,
       COALESCE((
           SELECT array_agg(DISTINCT r.subject_key)
           FROM span r
           WHERE r.closed_at IS NULL
             AND r.subject_kind = 'name'
             AND r.facet = 'resolution'
             AND r.is_gap = FALSE
             AND jsonb_typeof(r.value -> 'addresses') = 'array'
             AND r.value -> 'addresses' @> to_jsonb(s.subject_key)
       ), '{}'::text[])::text[] AS citers
FROM span s
JOIN cited c ON c.addr = s.subject_key
WHERE s.closed_at IS NULL
  AND s.subject_kind = 'address'
ORDER BY s.subject_key, s.id;

-- name: ListOpenSpansBeneathAddresses :many
-- LIKE only prefilters; Go re-parses each key, so a loose pattern closes no stranger (#1689).
SELECT s.id, s.subject_kind, s.subject_key
FROM span s
WHERE s.closed_at IS NULL
  AND s.subject_kind IN ('service', 'endpoint')
  AND EXISTS (
      SELECT 1 FROM unnest(sqlc.arg(addresses)::text[]) AS a(addr)
      WHERE s.subject_key LIKE a.addr || ':%'
         OR s.subject_key LIKE '[' || a.addr || ']:%'
         OR s.subject_key LIKE '%@' || a.addr || ':%'
         OR s.subject_key LIKE '%@[' || a.addr || ']:%'
  )
ORDER BY s.subject_kind, s.subject_key, s.id;

-- name: ListNameCitationSpansWithinCurrency :many
WITH cover AS (
    -- Each timeline's own currency bound, the one NameCitedAddresses reads (ADR-0044).
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    WHERE o.subject_kind = 'name' AND o.facet IN ('resolution', 'dns-record')
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
)
SELECT sp.subject_key, sp.vantage_id, sp.facet, sp.value, sp.opened_at, sp.closed_at
FROM span sp
JOIN cover c
    ON  c.subject_key   = sp.subject_key
    AND c.facet         = sp.facet
    AND c.discriminator = sp.discriminator
    AND c.vantage_id IS NOT DISTINCT FROM sp.vantage_id
    AND c.source        = sp.source
WHERE sp.subject_kind = 'name'
  AND sp.facet IN ('resolution', 'dns-record')
  AND NOT sp.is_gap
  AND sp.opened_at <= sqlc.arg(at)::timestamptz
  AND (sp.closed_at IS NULL
       OR EXTRACT(EPOCH FROM (sqlc.arg(at)::timestamptz - sp.closed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence)
ORDER BY sp.subject_key, sp.vantage_id, sp.facet, sp.opened_at, sp.id;
