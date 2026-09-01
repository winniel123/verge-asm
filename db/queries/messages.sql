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

-- name: ListReadMessageIDs :many
-- The ids of every message the caller has read (#327). The panel and Inbox render
-- a per-row read badge and an unread filter; both are per-account facts, so the
-- read path resolves them from this account's own read-marks rather than the
-- retired global message.read_at column. Returned as a set the handler indexes by
-- id while shaping each row.
SELECT message_id FROM message_read WHERE account_id = sqlc.arg(account_id);

-- name: CountUnreadMessages :one
-- The unread count the caller's nav element carries on every screen (#327).
-- Read-state is a per-account fact: a message is unread for THIS account until
-- THIS account has a message_read row for it. Counts messages the caller has not
-- yet marked read — never a global count, so one account's mark-all cannot clear
-- another account's badge.
SELECT count(*) FROM message m
WHERE NOT EXISTS (
    SELECT 1 FROM message_read mr
    WHERE mr.message_id = m.id AND mr.account_id = sqlc.arg(account_id)
);

-- name: MarkMessageRead :exec
-- Mark one message read by the caller at the given instant (#327). Writes a
-- per-account read-mark, never the global message.read_at. Idempotent: a second
-- mark leaves the account's first read instant in place (ON CONFLICT DO NOTHING),
-- since read-state is a fact about having seen it and does not move on a re-read.
INSERT INTO message_read (account_id, message_id, read_at)
VALUES (sqlc.arg(account_id), sqlc.arg(message_id), sqlc.arg(read_at))
ON CONFLICT (account_id, message_id) DO NOTHING;

-- name: MarkAllMessagesRead :exec
-- Mark every message the caller has not yet read as read by the caller (#327) —
-- the panel's "mark all read" affordance, now scoped to the caller. Inserts one
-- read-mark per still-unread message for this account only; other accounts' badges
-- are untouched. ON CONFLICT DO NOTHING guards against a concurrent single-mark.
INSERT INTO message_read (account_id, message_id, read_at)
SELECT sqlc.arg(account_id), m.id, sqlc.arg(read_at)
FROM message m
WHERE NOT EXISTS (
    SELECT 1 FROM message_read mr
    WHERE mr.message_id = m.id AND mr.account_id = sqlc.arg(account_id)
)
ON CONFLICT (account_id, message_id) DO NOTHING;

-- name: MarkMessageUnread :exec
-- Return one message to unread for the caller (#473, ADR-0116): clear this
-- account's read-mark so the message counts as unread again. Read-state is a
-- per-account fact held in message_read, so deleting only this account's row can
-- never touch another operator's badge. Idempotent: deleting an absent row is a
-- no-op, so re-marking an already-unread message is harmless. This is the inverse
-- of MarkMessageRead — the design's Inbox renders a "Mark unread" affordance
-- (Inbox.jsx:59), so read is reversible.
DELETE FROM message_read
WHERE account_id = sqlc.arg(account_id) AND message_id = sqlc.arg(message_id);

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

-- name: ListAddressExclusionWithdrawals :many
-- Every open timeline a DECLARED address exclusion withdraws, for the membership
-- fold to close with the `descoped` ground (ADR-0133 §8, #1032). It is the
-- listing twin of PreviewExclusionWithdrawal: the same two CTE shapes read
-- against the declared `exclusion` rows instead of one candidate CIDR, so the
-- preview counts and the withdrawal act over the same set by construction.
--
-- It reads the exclusion corpus itself rather than taking one CIDR per call. The
-- fold runs this once per batch, and once the withdrawal has closed the spans it
-- returns no row, so a later batch does no work and writes no second message.
--
-- The withdrawal is never larger than the declaration it narrows (ADR-0133 §1):
-- an address a current resolution still cites does NOT leave, which is the NOT
-- EXISTS clause. The SECOND survivor rule — an address the custody extension
-- still reaches does not leave either — is applied in Go, because it is
-- custody.Estate.Derive and the rejected-alternatives table forbids restating
-- that rule outside the package the corpus locks. So this query answers which
-- timelines are CANDIDATES to close, and composeAddressWithdrawals decides.
--
-- Two limits are inherited from PreviewExclusionWithdrawal deliberately, so the
-- receipt and the act cannot drift apart. The `~ '^[0-9.]+$'` gate reads IPv4
-- subject keys alone, so an IPv6 address exclusion previews and withdraws
-- nothing. And the resolution test is a substring match over the span value, so
-- a resolution citing 10.0.0.10 also holds 10.0.0.1 in the estate. Both bound
-- the withdrawal SMALLER than the model asks, never larger, which is the safe
-- direction: an address that should have left stays, and none leaves that should
-- have stayed. Widening either one has to move both queries together.
WITH withdrawn_addr AS (
    SELECT DISTINCT s.subject_key
    FROM span s
    WHERE s.closed_at IS NULL
      AND s.subject_kind = 'address'
      AND s.subject_key ~ '^[0-9.]+$'
      AND EXISTS (
          SELECT 1 FROM exclusion e
          WHERE e.kind = 'address'
            AND e.address_cidr IS NOT NULL
            AND s.subject_key::inet <<= e.address_cidr
      )
      AND NOT EXISTS (
          SELECT 1 FROM span r
          WHERE r.closed_at IS NULL
            AND r.facet = 'resolution'
            AND r.is_gap = false
            AND position(s.subject_key IN r.value::text) > 0
      )
)
SELECT s.id, s.subject_kind, s.subject_key
FROM span s
WHERE s.closed_at IS NULL
  AND (
      s.subject_key IN (SELECT subject_key FROM withdrawn_addr)
      OR (s.subject_kind IN ('service', 'endpoint')
          AND EXISTS (SELECT 1 FROM withdrawn_addr w WHERE s.subject_key LIKE w.subject_key || ':%'))
  )
ORDER BY s.subject_key, s.id;

-- name: ListPendingSeedWithdrawals :many
-- The tombstones of withdrawn address Seeds the membership fold has not spent yet
-- (ADR-0134 §2, #1040). Each row is the MOVER of one withdrawal: the CIDR the
-- operator stopped declaring, which is both the mover's identity and the site the
-- coverage message fires at.
--
-- A spent row is filtered out on `consumed_at`, never on `consumed_batch_id`: the
-- batch FK sets that id NULL if its batch ever goes, and reading the id would then
-- resurrect the tombstone and withdraw the same ground a second time.
SELECT w.id, w.address_cidr
FROM seed_withdrawal w
WHERE w.consumed_at IS NULL
ORDER BY w.id;

-- name: ListSeedWithdrawalCandidates :many
-- Every open timeline a pending Seed-withdrawal tombstone MAY withdraw, for the
-- membership fold to close with the `descoped` ground (ADR-0134 §5, #1040).
--
-- It is the tombstone twin of ListAddressExclusionWithdrawals and carries the same
-- two CTE shapes, read against `seed_withdrawal` instead of `exclusion`, so the
-- two narrowing acts remove the same shape of ground.
--
-- It answers CANDIDATES. Of ADR-0134 §4's three survivor rules this query applies
-- ONE — an address a current resolution still cites does not leave, the NOT EXISTS
-- clause. The other two are decided in Go by composeSeedWithdrawals: a LIVE Seed
-- covering the address (read from the Seed corpus, never from the tombstone, which
-- is what settles a second covering Seed and a re-declared scope), and
-- custody.Estate.Derive still calling the address `operator`, which the
-- rejected-alternatives table forbids restating outside the package the corpus
-- locks.
--
-- The two limits ListAddressExclusionWithdrawals inherits from
-- PreviewExclusionWithdrawal are inherited here too, so all three queries bound
-- the same set. The `~ '^[0-9.]+$'` gate reads IPv4 subject keys alone, and the
-- resolution test is a substring match over the span value. Both bound the
-- withdrawal SMALLER than the model asks, never larger: an address that should
-- have left stays, and none leaves that should have stayed.
WITH withdrawn_addr AS (
    SELECT DISTINCT s.subject_key
    FROM span s
    WHERE s.closed_at IS NULL
      AND s.subject_kind = 'address'
      AND s.subject_key ~ '^[0-9.]+$'
      AND EXISTS (
          SELECT 1 FROM seed_withdrawal w
          WHERE w.consumed_at IS NULL
            AND s.subject_key::inet <<= w.address_cidr
      )
      AND NOT EXISTS (
          SELECT 1 FROM span r
          WHERE r.closed_at IS NULL
            AND r.facet = 'resolution'
            AND r.is_gap = false
            AND position(s.subject_key IN r.value::text) > 0
      )
)
SELECT s.id, s.subject_kind, s.subject_key
FROM span s
WHERE s.closed_at IS NULL
  AND (
      s.subject_key IN (SELECT subject_key FROM withdrawn_addr)
      OR (s.subject_kind IN ('service', 'endpoint')
          AND EXISTS (SELECT 1 FROM withdrawn_addr w WHERE s.subject_key LIKE w.subject_key || ':%'))
  )
ORDER BY s.subject_key, s.id;

-- name: MarkSeedWithdrawalsConsumed :exec
-- Spends the tombstones a fold has acted on, stamping the batch that performed the
-- withdrawal (ADR-0134 §5). It runs after the closures, in the same batch
-- transaction, so a rolled-back fold spends nothing.
--
-- Every tombstone the fold READ is spent, including one that closed nothing. A
-- withdrawal over ground a live Seed still covers, or ground no open timeline
-- sits on, has taken everything it was going to take.
--
-- `WHERE consumed_at IS NULL` keeps the stamp write-once, so a row cannot be
-- re-attributed to a later batch.
UPDATE seed_withdrawal
SET consumed_at = sqlc.arg(consumed_at), consumed_batch_id = sqlc.arg(consumed_batch_id)
WHERE consumed_at IS NULL AND id = ANY(sqlc.arg(ids)::bigint[]);
