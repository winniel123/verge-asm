-- name: NextReportDeliveryNo :one
SELECT COALESCE(MAX(delivery_no), 0) + 1 AS next_no
FROM report_delivery
WHERE schedule_id = $1;

-- name: InsertReportDelivery :one
INSERT INTO report_delivery (schedule_id, period_start, period_end, delivery_no, state, delivered_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick;

-- name: TryInsertScheduledDelivery :one
INSERT INTO report_delivery (schedule_id, period_start, period_end, delivery_no, state, delivered_at, scheduled_tick)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (schedule_id, scheduled_tick) WHERE scheduled_tick IS NOT NULL DO NOTHING
RETURNING id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state;

-- name: MarkReportDeliveryDelivered :exec
UPDATE report_delivery
SET state = 'delivered', delivered_at = $2
WHERE id = $1;

-- name: GetLatestReportDelivery :one
SELECT id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick
FROM report_delivery
WHERE schedule_id = $1 AND state <> 'failed'
ORDER BY id DESC
LIMIT 1;

-- name: ListReportDeliveries :many
SELECT id, schedule_id, period_start, period_end, delivery_no, generated_at, delivered_at, state, scheduled_tick
FROM report_delivery
WHERE schedule_id = $1
ORDER BY id DESC;
