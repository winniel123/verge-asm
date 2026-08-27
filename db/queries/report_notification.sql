-- Reads and writes behind the report notify runner (P0.6c/T7, #508). A
-- report_notification is the Operational record of one link-only ready-message to a
-- Channel for a scheduled report run: it is NOT a Message and carries no estate
-- (ADR-0039, ADR-0081). It mirrors the delivery table's claim/retry/mark queries —
-- FOR UPDATE SKIP LOCKED, the shared queue.Backoff on retry, dead-letter on the spent
-- attempt budget — but keys on the report_delivery it announces, not a message, and
-- routes by the schedule's channel binding, not by class.

-- name: InsertReportNotification :exec
-- Enqueue one pending ready-message for a scheduled run (report_delivery_id) to its
-- schedule's bound Channel (channel_id). Called once per won tick in the dispatcher's
-- transaction, only when the schedule binds a channel — a download-only schedule
-- enqueues nothing. state defaults to 'pending' and run_after to now(), so the next
-- notify poll claims it.
INSERT INTO report_notification (report_delivery_id, channel_id)
VALUES ($1, $2);

-- name: ClaimReportNotification :one
-- The Postgres-backed claim: FOR UPDATE SKIP LOCKED over pending notifications whose
-- run_after has passed, oldest first, marking the winner 'sending' in one statement so
-- two workers never claim the same one. It joins the run and its schedule so the runner
-- has everything the link-only body needs — the report name and the run's period — plus
-- the channel to POST to and the attempt budget, in one read.
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
-- A 2xx: the ready-message reached the Channel. Clears the last error. The caller
-- flips the report_delivery receipt to 'delivered' in the same act.
UPDATE report_notification
SET state = 'delivered', last_error = NULL
WHERE id = $1;

-- name: RetryReportNotification :exec
-- A transient failure with attempts left: advance the attempt, push run_after out by
-- the shared backoff, and record the error. The row returns to 'pending' and the claim
-- index picks it up again once run_after passes. The receipt is never touched.
UPDATE report_notification
SET state = 'pending', attempt = $2, run_after = $3, last_error = $4
WHERE id = $1;

-- name: MarkReportNotificationUndelivered :exec
-- The attempt budget is spent: dead-letter the ready-message. The report_delivery
-- receipt is deliberately left 'generated' — the artifact was cut and stays viewable
-- in-instance; only the ready-message failed to leave (ADR-0039).
UPDATE report_notification
SET state = 'undelivered', last_error = $2
WHERE id = $1;
