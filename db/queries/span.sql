-- The drift engine's Span reads and writes (#190). The fold is incremental — one
-- completed Batch at a time (ADR-0007): for each observation's timeline it reads
-- the open span, and where the value or the Derivation vector moved it closes
-- that span and opens a new one. There is deliberately NO delete or compaction
-- query here — the Span corpus is never compacted (ADR-0041). A Transition and a
-- Break are derived on read from ListSpansForSubject's rows; neither is stored.
--
-- These reads are NOT routed through the live-tier observation gate (#237). The
-- gate makes the raw `observation` corpus structurally unreadable past a
-- timeline's live bound; these queries read `FROM span`, the already-derived
-- corpus the fold produced, which ADR-0041 keeps forever (never compacted). A Span
-- read is therefore not a re-derivation from a stale observation — the very thing
-- the gate exists to prevent — so applying an observation-tier `@as_of` bound here
-- would wrongly hide settled history rather than protect a derivation. The fold
-- (GetOpenSpan/OpenSpan/CloseSpan) consumes the just-completed Batch it is folding,
-- which is live by construction, so it needs no gate either.

-- name: GetOpenSpan :one
-- The one open span on a timeline, or no row where the timeline is new. vantage
-- and source are part of the key and vantage may be NULL (the shipped resolver
-- position carries no vantage row yet), so they are matched with IS NOT DISTINCT
-- FROM rather than =.
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
-- Open a new span for a timeline. The caller passes the canonical value, the
-- gap flag, the Derivation vector as a JSON array of {leaf,version}, and the id of
-- the Batch whose fold opened it (ADR-0111) — nullable, since a span opened outside
-- a batch fold cites none.
INSERT INTO span (
    subject_kind, subject_key, facet, discriminator, vantage_id, source,
    value, is_gap, derivation, opened_at, opened_batch_id
) VALUES (
    @subject_kind, @subject_key, @facet, @discriminator, sqlc.narg('vantage_id')::bigint,
    @source, @value, @is_gap, @derivation, @opened_at, sqlc.narg('opened_batch_id')::bigint
)
RETURNING id;

-- name: CloseSpan :exec
-- Close an open span at closed_at, recording a closure reason only where the
-- close is a withdrawal (reason is NULL for an ordinary value move or a version
-- change) and the id of the Batch whose fold closed it (ADR-0111) — nullable, since
-- a withdrawal closure is not a batch fold and cites none. A span is closed once and
-- never rewritten.
UPDATE span
SET closed_at = @closed_at,
    closure_reason = sqlc.narg('closure_reason'),
    closed_batch_id = sqlc.narg('closed_batch_id')::bigint
WHERE id = @id AND closed_at IS NULL;

-- name: ListOpenSpansForSubject :many
-- Every open timeline a subject currently holds — what a withdrawal closes, all
-- at once, with the ground it rests on.
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE subject_kind = @subject_kind AND subject_key = @subject_key AND closed_at IS NULL
ORDER BY facet, discriminator, vantage_id, source;

-- name: ListAllOpenSpans :many
-- Every open span across the whole estate — the Inventory axis read (#243,
-- ADR-0105). The span_open_timeline_idx guarantees at most one open span per
-- (subject, facet, discriminator, vantage, source) timeline, so each row IS the
-- value that timeline currently holds — the estate's inventory, read straight off
-- the derived corpus with no re-derivation. A withdrawal closes a timeline's span
-- (ADR-0082), so an open span is a current member by construction; there is no
-- membership re-derivation and no denominator here, exactly as the Subjects
-- listing states none (ADR-0072). Gaps are included: a Gap is a facet the system
-- currently cannot value, and inventory states that rather than hiding it. Like
-- the other span reads this is NOT live-tier gated — it reads the already-derived,
-- never-compacted `span` corpus (ADR-0041), not the observation tier. Ordered by
-- subject so the renderer groups a subject's facets in a single pass.
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE closed_at IS NULL
ORDER BY subject_kind, subject_key, facet, discriminator, vantage_id, source;

-- name: ListSpansOpenSince :many
-- Every span that was open at any instant from @since onward — still open now, or
-- closed after @since. This is exactly the corpus a vs-last-batch delta needs
-- (P0.2, design-system PARITY-CHART.md): the currently-open population AND the
-- spans the most recent batch closed, so the population open at the previous batch
-- boundary is reconstructable on read (internal/drift.OpenAt) alongside the current
-- one. Passing the previous batch's instant as @since keeps the scan to recent
-- drift rather than the whole never-compacted corpus. Like the other span reads it
-- is NOT live-tier gated — it reads the already-derived `span` corpus (ADR-0041),
-- not the observation tier. Ordered by subject so a per-subject fold is one pass.
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE closed_at IS NULL OR closed_at > @since
ORDER BY subject_kind, subject_key, facet, discriminator, vantage_id, source, opened_at;

-- name: ListServiceReachabilitySpansByClassAt :many
-- The `reachability` span per (Service, Vantage class) that was OPEN at instant @at
-- — the as-of-a-past-batch twin of ListServiceReachabilitySpansByClass, for the
-- Exposure stat band's vs-last-batch deltas (P0.2). It reconstructs each leg's
-- value as it stood at @at from the never-compacted span corpus (ADR-0041): a span
-- open at @at has opened_at <= @at and had not yet closed (still open, or closed
-- after @at). DISTINCT ON keeps the most recent such span per (service, class),
-- exactly as the current read keeps the most recent open one. Class is the static
-- vantage column (the same join the current read uses); the exposure projection is
-- computed in the handler over both readings. NOT live-tier gated (span corpus).
SELECT DISTINCT ON (sp.subject_key, v.class)
    sp.subject_key AS subject_key,
    v.class        AS class,
    sp.value       AS value,
    sp.is_gap      AS is_gap
FROM span sp
JOIN vantage v ON v.id = sp.vantage_id
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.opened_at <= @at
  AND (sp.closed_at IS NULL OR sp.closed_at > @at)
ORDER BY sp.subject_key, v.class, sp.opened_at DESC, sp.id DESC;

-- name: ListSpansForSubject :many
-- A subject's full Span history — current and closed — for the Subjects
-- drill-down. Ordered by timeline, oldest first, so the renderer walks each
-- timeline and derives its Breaks and Transitions on read. The closed corpus is
-- never compacted, so a withdrawn Name's closed timelines render in full.
SELECT id, subject_kind, subject_key, facet, discriminator, vantage_id, source,
       value, is_gap, derivation, opened_at, closed_at, closure_reason
FROM span
WHERE subject_kind = @subject_kind AND subject_key = @subject_key
ORDER BY facet, discriminator, vantage_id, source, opened_at, id;

-- name: ListRecentDriftEvents :many
-- The estate-wide, batch-grouped drift feed (#288, ADR-0111). Every span open/close
-- EVENT a Batch caused within the period, joined to that Batch for the group meta, so
-- the handler derives each event's change kind on read (ADR-0007) and groups the
-- transitions by batch. Two event roles are unioned:
--
--   'opened' — a span opened by the batch (opened_batch_id): the anchor for
--   appeared / returned / revealed / changed. Its predecessor span on the same
--   timeline (the most recent span opened before it) rides along so the handler can
--   classify the opening and build a `changed` transition's before/after diff.
--
--   'closed' — a span closed by the batch WITH a closure reason (closed_batch_id +
--   closure_reason): the anchor for withdrawn / descoped. An ordinary value-move
--   close carries no reason and is already represented by its successor's 'opened'
--   row, so it is excluded here to avoid counting the same transition twice.
--
-- Reads span and batch only — never dispatch — honoring the comparison-path
-- separation (ADR-0041). Ordered newest batch first, then by timeline for a stable
-- per-batch render.
SELECT
    'opened'::text   AS role,
    b.id             AS batch_id,
    b.kind           AS batch_kind,
    b.created_at     AS batch_at,
    b.recorded_scope AS recorded_scope,
    sp.subject_kind, sp.subject_key, sp.facet, sp.discriminator,
    sp.value, sp.is_gap, sp.derivation,
    sp.opened_at, sp.closed_at, sp.closure_reason,
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
    NULL::jsonb        AS prev_value,
    NULL::jsonb        AS prev_derivation,
    NULL::timestamptz  AS prev_closed_at,
    NULL::text         AS prev_closure_reason
FROM span sp
JOIN batch b ON b.id = sp.closed_batch_id
WHERE b.created_at >= @since
  AND sp.closure_reason IS NOT NULL

ORDER BY batch_at DESC, batch_id DESC, subject_kind, subject_key, facet, discriminator, opened_at
LIMIT @max_events;

-- name: ListWithdrawalLifespans :many
-- Every subject withdrawal since @since, paired with the subject's first appearance,
-- so the web layer derives the mean-time-to-withdrawal trend (P0.3, #444). A
-- withdrawal closes EVERY open timeline a subject held at one instant (ADR-0082,
-- CloseWithdrawal), so the per-facet closures collapse to one subject departure:
-- DISTINCT ON (subject_kind, subject_key, closed_at) keeps one row per departure.
-- first_opened is the earliest opened_at across ALL the subject's spans — its
-- appearance — so time-to-withdrawal is withdrawn_at - first_opened. Only a WITHDRAWAL
-- close counts: closure_reason IS NOT NULL excludes an ordinary value-move close
-- (which carries no reason and is not a departure). Reads FROM span only — the
-- already-derived, never-compacted corpus (ADR-0041) — so it is NOT live-tier gated;
-- an @as_of bound would wrongly hide settled history rather than protect a
-- re-derivation. Ordered by the withdrawal instant for a stable, oldest-first series.
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
-- Every Name/Service subject whose FIRST appearance is at or after @since, paired
-- with that first-appearance instant — the corpus the Reports "New assets
-- discovered" card folds into a per-period count and a daily-discovery series
-- (P2.4b, #468). A subject's appearance is the earliest opened_at across ALL its
-- spans (the `appeared` drift classification): GROUP BY collapses a subject's many
-- facet timelines to that one instant, and HAVING keeps only subjects that first
-- appeared in the window, so a subject long-present before @since is not miscounted
-- as newly discovered. Only Name and Service subjects are counted — the same
-- watched population the assets-watched census reads (internal/drift.DistinctSubjects)
-- — so an Endpoint or Address facet moving is not itself a new asset. Reads FROM
-- span only — the already-derived, never-compacted corpus (ADR-0041) — so it is NOT
-- live-tier gated; an @as_of bound would wrongly hide settled history rather than
-- protect a re-derivation. Ordered by the appearance instant for a stable,
-- oldest-first fold.
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
-- The open `Service` population the weekly `tls-acceptance` Scan enumerates over
-- (#199, ADR-0028): every Service whose CURRENT `reachability` span reads `reached`,
-- with the vantage it was reached from. This is an enumeration over open Services,
-- NOT a port list — the ports are whatever the Services are open on, inherited from
-- `reachability` — so the Scan consults no port tier at all. A closed or gap span is
-- excluded: `tls-acceptance` is attempted only against a Service known open, the same
-- way the `certificate` handshake rides a reached connect. vantage_id is part of the
-- key and may be NULL (the shipped position carries no vantage row), carried through
-- so the fan-out partitions per vantage exactly as reachability does.
SELECT sp.subject_key AS service_key, sp.vantage_id AS vantage_id
FROM span sp
WHERE sp.subject_kind = 'service'
  AND sp.facet = 'reachability'
  AND sp.closed_at IS NULL
  AND sp.is_gap = FALSE
  AND (sp.value ->> 'outcome') = 'reached'
ORDER BY sp.vantage_id, sp.subject_key;

-- name: ListReachabilitySpansForExposure :many
-- The two most recent `reachability` spans per (Service, vantage), joined to the
-- vantage's prober endpoint — the Exposure landing view's read (#196). rn = 1 is
-- the current span (the leg's value) and rn = 2 is its immediate predecessor
-- (the flagship internet not-reached -> reached transition is read from the pair).
-- Vantage class is deliberately NOT selected: the caller re-verifies it every
-- render from the prober's presented (dialled) address against the operator's
-- declared address scopes (CONTEXT.md `Vantage class`), never from a static
-- column, so this read carries the host and availability instead.
SELECT subject_key, vantage_id, host, availability, value, is_gap, derivation, opened_at, closed_at, rn
FROM (
    SELECT sp.subject_key AS subject_key,
           sp.vantage_id  AS vantage_id,
           v.host         AS host,
           v.availability AS availability,
           sp.value       AS value,
           sp.is_gap      AS is_gap,
           sp.derivation  AS derivation,
           sp.opened_at   AS opened_at,
           sp.closed_at   AS closed_at,
           ROW_NUMBER() OVER (
               PARTITION BY sp.subject_key, sp.vantage_id
               ORDER BY sp.opened_at DESC, sp.id DESC
           ) AS rn
    FROM span sp
    LEFT JOIN vantage v ON v.id = sp.vantage_id
    WHERE sp.subject_kind = 'service' AND sp.facet = 'reachability'
) ranked
WHERE rn <= 2
ORDER BY subject_key, vantage_id, rn;
