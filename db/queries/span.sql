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
-- gap flag, and the Derivation vector as a JSON array of {leaf,version}.
INSERT INTO span (
    subject_kind, subject_key, facet, discriminator, vantage_id, source,
    value, is_gap, derivation, opened_at
) VALUES (
    @subject_kind, @subject_key, @facet, @discriminator, sqlc.narg('vantage_id')::bigint,
    @source, @value, @is_gap, @derivation, @opened_at
)
RETURNING id;

-- name: CloseSpan :exec
-- Close an open span at closed_at, recording a closure reason only where the
-- close is a withdrawal (reason is NULL for an ordinary value move or a version
-- change). A span is closed once and never rewritten.
UPDATE span
SET closed_at = @closed_at, closure_reason = sqlc.narg('closure_reason')
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
