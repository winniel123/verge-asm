-- name: InsertMessage :one
INSERT INTO message (cause, class, subject_kind, fired_at, instant, census, headline)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at;

-- name: ListMessages :many
SELECT id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at
FROM message
ORDER BY id DESC;

-- name: ListReadMessageIDs :many
SELECT message_id FROM message_read WHERE account_id = sqlc.arg(account_id);

-- name: CountUnreadMessages :one
SELECT count(*) FROM message m
WHERE NOT EXISTS (
    SELECT 1 FROM message_read mr
    WHERE mr.message_id = m.id AND mr.account_id = sqlc.arg(account_id)
);

-- name: MarkMessageRead :exec
-- A re-read is not a new fact, so the first read instant stands.
INSERT INTO message_read (account_id, message_id, read_at)
VALUES (sqlc.arg(account_id), sqlc.arg(message_id), sqlc.arg(read_at))
ON CONFLICT (account_id, message_id) DO NOTHING;

-- name: MarkAllMessagesRead :exec
INSERT INTO message_read (account_id, message_id, read_at)
SELECT sqlc.arg(account_id), m.id, sqlc.arg(read_at)
FROM message m
WHERE NOT EXISTS (
    SELECT 1 FROM message_read mr
    WHERE mr.message_id = m.id AND mr.account_id = sqlc.arg(account_id)
)
ON CONFLICT (account_id, message_id) DO NOTHING;

-- name: MarkMessageUnread :exec
DELETE FROM message_read
WHERE account_id = sqlc.arg(account_id) AND message_id = sqlc.arg(message_id);

-- name: PreviewExclusionWithdrawal :one
-- IPv4 subject keys only, so an IPv6 exclusion previews and withdraws nothing.
-- A substring resolution test, so a resolution citing 10.0.0.10 also holds 10.0.0.1.
-- Both bound this smaller than the model asks, never larger; widening either moves every copy.
WITH cidr AS (
    SELECT sqlc.arg(cidr)::cidr AS net
),
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
SELECT w.id, w.address_cidr
FROM seed_withdrawal w
  -- The batch FK nulls consumed_batch_id, so filtering on it would resurrect a spent tombstone.
WHERE w.consumed_at IS NULL
  AND w.kind = 'address'
ORDER BY w.id
  -- Unclaimed, two concurrent folds each write the coverage message permanently (ADR-0134 §5.1).
FOR UPDATE SKIP LOCKED;

-- name: ListSeedWithdrawalCandidates :many
WITH withdrawn_addr AS (
    SELECT DISTINCT s.subject_key
    FROM span s
    WHERE s.closed_at IS NULL
      AND s.subject_kind = 'address'
      AND s.subject_key ~ '^[0-9.]+$'
      AND EXISTS (
          SELECT 1 FROM unnest(sqlc.arg(cidrs)::text[]) AS w(net)
          WHERE s.subject_key::inet <<= w.net::cidr
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

-- name: SpendSeedWithdrawals :exec
UPDATE seed_withdrawal w
SET consumed_at = sqlc.arg(consumed_at), consumed_batch_id = sqlc.arg(consumed_batch_id)
WHERE w.consumed_at IS NULL
  AND w.kind = 'address'
  AND w.id = ANY(sqlc.arg(ids)::bigint[])
  -- Spending an IPv6 tombstone loses ground the gate cannot see, mover and all (ADR-0134 §5.1).
  AND family(w.address_cidr) = 4
  AND NOT EXISTS (
      SELECT 1 FROM span s
      WHERE s.closed_at IS NULL
        AND s.subject_kind = 'address'
        AND s.subject_key ~ '^[0-9.]+$'
        AND s.subject_key::inet <<= w.address_cidr
  );

-- name: ListPendingNameSeedWithdrawals :many
SELECT w.id, w.name_domain
FROM seed_withdrawal w
WHERE w.consumed_at IS NULL
  AND w.kind = 'name'
ORDER BY w.id
FOR UPDATE SKIP LOCKED;

-- name: ListNameSeedWithdrawalCandidates :many
SELECT s.id, s.subject_key
FROM span s
WHERE s.closed_at IS NULL
  AND s.subject_kind = 'name'
  AND EXISTS (
      SELECT 1 FROM unnest(sqlc.arg(domains)::text[]) AS w(domain)
      WHERE s.subject_key = w.domain OR s.subject_key LIKE '%.' || w.domain
  )
ORDER BY s.subject_key, s.id;

-- name: SpendNameSeedWithdrawals :exec
-- Every dns job waits, not only older ones: a retry re-enqueues the frozen spec (ADR-0135 §5).
UPDATE seed_withdrawal w
SET consumed_at = sqlc.arg(consumed_at), consumed_batch_id = sqlc.arg(consumed_batch_id)
WHERE w.consumed_at IS NULL
  AND w.kind = 'name'
  AND w.id = ANY(sqlc.arg(ids)::bigint[])
  AND (
      EXISTS (
          SELECT 1 FROM seed s
          WHERE s.kind = 'name'
            AND s.name_domain IS NOT NULL
            AND (w.name_domain = s.name_domain OR w.name_domain LIKE '%.' || s.name_domain)
      )
      OR (
          NOT EXISTS (
              SELECT 1 FROM span s
              WHERE s.closed_at IS NULL
                AND s.subject_kind = 'name'
                AND (s.subject_key = w.name_domain OR s.subject_key LIKE '%.' || w.name_domain)
          )
          AND NOT EXISTS (
              SELECT 1 FROM queue_job j
              WHERE j.kind = 'resolution-walk'
                AND j.state IN ('ready', 'running')
          )
      )
  );
