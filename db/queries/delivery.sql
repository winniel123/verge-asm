-- Reads and writes behind Channel delivery (#207). A Delivery is the Operational
-- record of one outbound POST of one Message to one Channel: it never becomes a
-- Message and never touches the comparison path. Routing is by class alone — the
-- only predicate over which channels receive a firing is the class subset each
-- channel carries (ADR-0091); there is no per-rule or per-subject query here.

-- name: InsertDelivery :exec
-- Enqueue one pending Delivery for (message, channel). The caller has already
-- decided membership by class alone (delivery.Routes); this only persists the
-- routed pair. Idempotent: re-enqueuing the same pair is a no-op, so the message
-- identifier the receiver de-duplicates on is stable across retries.
INSERT INTO delivery (message_id, channel_id)
VALUES ($1, $2)
ON CONFLICT ON CONSTRAINT delivery_message_channel DO NOTHING;

-- name: ClaimDelivery :one
-- The Postgres-backed claim: FOR UPDATE SKIP LOCKED over pending deliveries whose
-- run_after has passed, oldest first, marking the winner 'sending' in one
-- statement so two workers never claim the same delivery.
UPDATE delivery SET state = 'sending', updated_at = now()
WHERE id = (
    SELECT id FROM delivery
    WHERE state = 'pending' AND run_after <= now()
    ORDER BY run_after, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, message_id, channel_id, attempt, max_attempts;

-- name: GetMessageForDelivery :one
-- The frozen Message the body is built from — read verbatim, never recomputed.
-- The body carries exactly these fields (the headline byte-identical, the census
-- as a count) and reaches no other table: no row behind a census count.
SELECT id, cause, class, subject_kind, fired_at, instant, census, headline
FROM message
WHERE id = $1;

-- name: GetChannelForDelivery :one
-- Reads the target URL and the signing secret. This is the ONE read path that
-- selects the secret: it is write-only at the interface (no render query returns
-- it), but the worker-side signer must read it to compute the HMAC. It is never
-- rendered and never leaves this instance except as a signature.
SELECT url, secret FROM channel WHERE id = $1;

-- name: MarkDeliveryDelivered :exec
-- A 2xx: the delivery is complete. Clears the last error and stamps the instant.
UPDATE delivery
SET state = 'delivered', delivered_at = $2, last_error = NULL, updated_at = now()
WHERE id = $1;

-- name: RetryDelivery :exec
-- A transient failure with attempts left: advance the attempt, push run_after out
-- by the shared backoff, and record the error. The row returns to 'pending' and
-- the claim index picks it up again once run_after passes.
UPDATE delivery
SET state = 'pending', attempt = $2, run_after = $3, last_error = $4, updated_at = now()
WHERE id = $1;

-- name: MarkDeliveryUndelivered :exec
-- The attempt budget is spent: dead-letter. The row is marked 'undelivered' — the
-- undelivered mark — and the Message it points at is deliberately left untouched.
UPDATE delivery
SET state = 'undelivered', last_error = $2, updated_at = now()
WHERE id = $1;

-- name: ListDeliveriesForMessage :many
-- The delivery outcomes a Message renders in the store (notification-channels.md
-- §8): to which channels it went and whether any is dead-lettered. Reads from the
-- delivery table by join — the Message row carries no delivery state of its own.
SELECT d.id, d.channel_id, c.url, d.state, d.attempt, d.last_error, d.delivered_at
FROM delivery d
JOIN channel c ON c.id = d.channel_id
WHERE d.message_id = $1
ORDER BY d.id;

-- name: ListDeliveryOutcomes :many
-- Every Delivery outcome joined to its Channel, for rendering each Message's own
-- delivery outcomes on the panel in one pass (ADR-0081, ADR-0039) rather than a
-- per-message read. A delivery failure is surfaced HERE, on the Message it
-- carries — never on Coverage, which a delivery has no cause to touch (#244).
-- Ordered by message then delivery so the caller groups them in a single walk.
SELECT d.message_id, d.channel_id, c.url, d.state, d.attempt, d.last_error
FROM delivery d
JOIN channel c ON c.id = d.channel_id
ORDER BY d.message_id, d.id;
