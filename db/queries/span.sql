-- The drift engine's Span reads and writes (#190). The fold is incremental — one
-- completed Batch at a time (ADR-0007): for each observation's timeline it reads
-- the open span, and where the value or the Derivation vector moved it closes
-- that span and opens a new one. There is deliberately NO delete or compaction
-- query here — the Span corpus is never compacted (ADR-0041). A Transition and a
-- Break are derived on read from ListSpansForSubject's rows; neither is stored.

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
