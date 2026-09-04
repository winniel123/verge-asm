-- name: InsertReportNotification :exec
INSERT INTO report_notification (report_delivery_id, channel_id)
VALUES ($1, $2);

-- name: ClaimReportNotification :one
UPDATE report_notification n
SET state = 'sending'
FROM report_delivery d, report_schedule s
WHERE n.id = (
    SELECT id FROM report_notification
    WHERE state = 'pending' AND run_after <= now()
    ORDER BY run_after, id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
  AND d.id = n.report_delivery_id
  AND s.id = d.schedule_id
RETURNING n.id, n.report_delivery_id, n.channel_id, n.attempt, n.max_attempts,
          d.period_start, d.period_end, s.name;

-- name: MarkReportNotificationDelivered :exec
UPDATE report_notification
SET state = 'delivered', last_error = NULL
WHERE id = $1;

-- name: RetryReportNotification :exec
UPDATE report_notification
SET state = 'pending', attempt = $2, run_after = $3, last_error = $4
WHERE id = $1;

-- name: MarkReportNotificationUndelivered :exec
UPDATE report_notification
SET state = 'undelivered', last_error = $2
WHERE id = $1;
