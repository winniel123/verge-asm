-- Reads and writes behind the global message panel (#205). A Message is one
-- firing of one cause, written once at the cause and never recomputed
-- (CONTEXT.md `Message`, ADR-0064). The store is unconditional — there is no
-- enable, no routing and no way to turn it off — so there is a plain insert, an
-- unbounded newest-first list, an unread count for the nav element, and a
-- read-state toggle. There is deliberately no update-of-content and no delete:
-- a message is written once and retained while the operator may still read it.

-- name: InsertMessage :one
-- Write one computed message. The caller has already decided the cause, class,
-- fired-at key, instant, census and headline at the cause; this only persists
-- them. census is NULL where the firing carries a count rather than rows.
INSERT INTO message (cause, class, subject_kind, fired_at, instant, census, headline)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at;

-- name: ListMessages :many
-- Every message, newest-first, unbounded — no cap or load-more ships, since no
-- install has yet accumulated enough live volume to say whether one is needed
-- (v1 spec §6.7, #160). The panel renders each row linking per its mover.
SELECT id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at
FROM message
ORDER BY id DESC;

-- name: CountUnreadMessages :one
-- The unread count the global nav element carries on every screen. Reads the
-- partial index over unread rows.
SELECT count(*) FROM message WHERE read_at IS NULL;

-- name: MarkMessageRead :exec
-- Mark one message read at the given instant. Idempotent: marking an already-read
-- message leaves its first read instant in place, since read-state is a fact
-- about the operator having seen it and does not move on a second view.
UPDATE message SET read_at = $2 WHERE id = $1 AND read_at IS NULL;

-- name: MarkAllMessagesRead :exec
-- Mark every unread message read — the panel's "mark all read" affordance.
UPDATE message SET read_at = $1 WHERE read_at IS NULL;

-- name: PreviewExclusionWithdrawal :one
-- The honestly-computable narrowing receipt (#205 AC8, ADR-0074): count the
-- subjects a candidate ADDRESS exclusion would withdraw and the timelines they
-- take out of the estate, read from the live span corpus rather than fabricated.
-- A narrowing withdraws only ground nothing else cites — a subject a current
-- resolution still holds survives, and its `Gap` carries it, so it is NOT
-- counted here (the NOT EXISTS clause). The preview fires only where the count is
-- non-zero. Scoped to address exclusions, the one narrowing whose withdrawn set
-- the prototype (#167) demonstrates firing; a name/subtree exclusion whose names
-- still resolve is the survives-via-Gap case and returns zero.
WITH cidr AS (
    SELECT sqlc.arg(cidr)::cidr AS net
),
-- The address subjects the exclusion removes: an IPv4 address inside the excluded
-- scope, currently in the estate (an open span), whose membership no current
-- resolution still holds.
withdrawn_addr AS (
    SELECT DISTINCT s.subject_key
    FROM span s, cidr
    WHERE sqlc.arg(kind)::text = 'address'
      AND s.closed_at IS NULL
      AND s.subject_kind = 'address'
      AND s.subject_key ~ '^[0-9.]+$'
      AND s.subject_key::inet <<= cidr.net
      AND NOT EXISTS (
          SELECT 1 FROM span r
          WHERE r.closed_at IS NULL
            AND r.facet = 'resolution'
            AND r.is_gap = false
            AND position(s.subject_key IN r.value::text) > 0
      )
),
-- Every subject the withdrawal takes with it: the addresses and the Services and
-- Endpoints sitting on them (their keys carry the address as a prefix).
withdrawn_subject AS (
    SELECT s.subject_key
    FROM span s
    WHERE s.closed_at IS NULL
      AND (
          s.subject_key IN (SELECT subject_key FROM withdrawn_addr)
          OR (s.subject_kind IN ('service', 'endpoint')
              AND EXISTS (SELECT 1 FROM withdrawn_addr w WHERE s.subject_key LIKE w.subject_key || ':%'))
      )
    GROUP BY s.subject_key
),
withdrawn_span AS (
    SELECT s.id
    FROM span s
    WHERE s.closed_at IS NULL
      AND (
          s.subject_key IN (SELECT subject_key FROM withdrawn_addr)
          OR (s.subject_kind IN ('service', 'endpoint')
              AND EXISTS (SELECT 1 FROM withdrawn_addr w WHERE s.subject_key LIKE w.subject_key || ':%'))
      )
)
SELECT
    (SELECT count(*) FROM withdrawn_subject)::bigint AS subjects_withdrawn,
    (SELECT count(*) FROM withdrawn_span)::bigint   AS timelines_removed;
