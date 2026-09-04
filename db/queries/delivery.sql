-- name: InsertDelivery :exec
INSERT INTO delivery (message_id, channel_id)
VALUES ($1, $2)
ON CONFLICT ON CONSTRAINT delivery_message_channel DO NOTHING;

-- name: ClaimDelivery :one
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
SELECT id, cause, class, subject_kind, fired_at, instant, census, headline
FROM message
WHERE id = $1;

-- name: GetChannelForDelivery :one
SELECT url, secret FROM channel WHERE id = $1;

-- name: MarkDeliveryDelivered :exec
UPDATE delivery
SET state = 'delivered', delivered_at = $2, last_error = NULL, updated_at = now()
WHERE id = $1;

-- name: RetryDelivery :exec
UPDATE delivery
SET state = 'pending', attempt = $2, run_after = $3, last_error = $4, updated_at = now()
WHERE id = $1;

-- name: MarkDeliveryUndelivered :exec
UPDATE delivery
SET state = 'undelivered', last_error = $2, updated_at = now()
WHERE id = $1;

-- name: ListDeliveriesForMessage :many
SELECT d.id, d.channel_id, c.url, d.state, d.attempt, d.last_error, d.delivered_at
FROM delivery d
JOIN channel c ON c.id = d.channel_id
WHERE d.message_id = $1
ORDER BY d.id;

-- name: ListDeliveryOutcomes :many
SELECT d.message_id, d.channel_id, c.url, d.state, d.attempt, d.last_error
FROM delivery d
JOIN channel c ON c.id = d.channel_id
ORDER BY d.message_id, d.id;
