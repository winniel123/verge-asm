-- name: InsertMessage :one
-- A held row states the batch it waits on and the basis its census is read from (ADR-1806 §2).
INSERT INTO message (cause, class, subject_kind, fired_at, instant, census, headline, census_pending_after_batch, census_basis)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at, census_pending_after_batch, census_basis;

-- name: ListMessages :many
SELECT id, cause, class, subject_kind, fired_at, instant, census, headline, read_at, created_at, census_pending_after_batch, census_basis
FROM message
  -- A held row carries no census yet, so no operator surface may render it (ADR-1806 §2).
WHERE census_pending_after_batch IS NULL
ORDER BY id DESC;

-- name: ListReadMessageIDs :many
SELECT message_id FROM message_read WHERE account_id = sqlc.arg(account_id);

-- name: CountUnreadMessages :one
SELECT count(*) FROM message m
  -- A held row counts toward no unread badge (ADR-1806 §2).
WHERE m.census_pending_after_batch IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM message_read mr
    WHERE mr.message_id = m.id AND mr.account_id = sqlc.arg(account_id)
);

-- name: MarkMessageRead :exec
-- A re-read is not a new fact, so the first read instant stands.
INSERT INTO message_read (account_id, message_id, read_at)
SELECT sqlc.arg(account_id), m.id, sqlc.arg(read_at)
FROM message m
  -- A row released later must still show unread, so the mark passes over it (ADR-1806 §2).
WHERE m.id = sqlc.arg(message_id)
  AND m.census_pending_after_batch IS NULL
ON CONFLICT (account_id, message_id) DO NOTHING;

-- name: MarkAllMessagesRead :exec
INSERT INTO message_read (account_id, message_id, read_at)
SELECT sqlc.arg(account_id), m.id, sqlc.arg(read_at)
FROM message m
  -- A row released later must still show unread, so the mark passes over it (ADR-1806 §2).
WHERE m.census_pending_after_batch IS NULL
  AND NOT EXISTS (
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

-- name: ListReleasableHeldMessages :many
-- A held row releases on a drained hot dispatch, or where the tier cannot answer (ADR-1806 §6).
SELECT m.id, m.class, m.headline, m.census_pending_after_batch, m.census_basis
FROM message m
JOIN batch b ON b.id = m.census_pending_after_batch
WHERE m.census_pending_after_batch IS NOT NULL
  AND (
      -- No reaper leaves the drain test reading a job set nothing reaps (ADR-1806 §6).
      sqlc.arg(reaper_disabled)::boolean
      -- A disabled hot tier opens nothing beneath the root, ever (ADR-1806 §6).
      OR NOT EXISTS (
          SELECT 1 FROM scan hs WHERE hs.kind = 'hot' AND hs.enabled
      )
      OR EXISTS (
          SELECT 1
          FROM (
              SELECT d.id
              FROM dispatch d
              JOIN scan s ON s.id = d.scan_id
              WHERE s.kind = 'hot'
                -- A skipped tick enqueues no job, so a drain test reads it drained (ADR-1806 §2).
                AND d.status = 'fanned-out'
                AND d.created_at >= b.created_at
              ORDER BY d.created_at, d.id
              LIMIT 1
          ) first_hot
            -- hot commits its dispatch row before its jobs, so an empty set is not drained (#1816).
          WHERE EXISTS (
              SELECT 1 FROM queue_job j
              WHERE j.dispatch_id = first_hot.id
          )
          AND NOT EXISTS (
              -- The cadence-lag gate's query excludes this dispatch's own jobs (ADR-1806 §3).
              SELECT 1 FROM queue_job j
              WHERE j.dispatch_id = first_hot.id
                AND j.state IN ('ready', 'running')
          )
      )
  )
ORDER BY m.id;

-- name: ReleaseHeldMessage :execrows
-- The guarded update is the claim: a second pass takes no row and owes no delivery (ADR-1806 §2).
UPDATE message
SET census = sqlc.arg(census),
    headline = sqlc.arg(headline),
    census_pending_after_batch = NULL
  -- A released row keeps the basis its census was computed from (25600_message_census_basis.sql).
WHERE id = sqlc.arg(id) AND census_pending_after_batch IS NOT NULL;
