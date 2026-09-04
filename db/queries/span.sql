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

-- name: ListSpansForSubject :many
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE subject_kind = @subject_kind AND subject_key = @subject_key
ORDER BY facet, discriminator, vantage_id, source, opened_at, id;

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
    pred.closure_reason AS prev_closure_reason
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
      AND p.opened_at < sp.opened_at
    ORDER BY p.opened_at DESC, p.id DESC
    LIMIT 1
) pred ON true
WHERE b.created_at >= @since

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
    NULL::text         AS prev_closure_reason
FROM span sp
JOIN batch b ON b.id = sp.closed_batch_id
WHERE b.created_at >= @since
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
