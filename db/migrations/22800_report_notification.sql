-- +goose Up
-- A report_notification is the Operational record of one link-only ready-message sent
-- to a Channel when a scheduled report_delivery is cut (P0.6c/T7, #508). It is NOT a
-- Message and carries no cause and no estate: the ready-message names the report, its
-- period, and a session-authed link to the in-instance artifact, and nothing more
-- (ADR-0039, ADR-0081). A report run is not a Message, so this is not modelled on the
-- delivery table (which keys on message_id and routes by class) — it keys on the
-- report_delivery it announces and the channel it announces to, with no class axis.
--
-- It mirrors the delivery table's retry/backoff/dead-letter shape exactly (#188):
-- pending → sending → delivered / undelivered, an attempt budget, and a run_after the
-- shared queue.Backoff pushes out on each retry. The retry curve is the queue's own,
-- not a second schedule beside it.
--
-- Crucially, a notify FAILURE never marks the report_delivery receipt 'failed': the
-- artifact was generated and stays viewable in-instance regardless of whether its
-- ready-message reached the Channel. Only this row records the send outcome. On
-- SUCCESS the caller flips the receipt to 'delivered' and stamps delivered_at; on
-- dead-letter the receipt stays 'generated' — the artifact is still there to open.
CREATE TABLE report_notification (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- The scheduled run this ready-message announces. NOT NULL: a notification exists
    -- only because a report_delivery was cut for a channel-bound schedule.
    report_delivery_id BIGINT NOT NULL REFERENCES report_delivery (id),
    -- The Channel the ready-message is POSTed to (the schedule's bound channel at the
    -- time of the run). NOT NULL: a download-only schedule enqueues no notification.
    channel_id         BIGINT NOT NULL REFERENCES channel (id),
    -- pending: attempts remain and run_after gates the next try.
    -- sending:  claimed by the notify runner, in flight.
    -- delivered: a 2xx was received; the receipt was flipped to 'delivered'.
    -- undelivered: dead-lettered after the attempt budget — the artifact stays viewable.
    state              TEXT NOT NULL DEFAULT 'pending'
                       CHECK (state IN ('pending', 'sending', 'delivered', 'undelivered')),
    attempt            INT NOT NULL DEFAULT 0,
    max_attempts       INT NOT NULL DEFAULT 5,
    run_after          TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- The last error string, channel-surface drill-down only. NULL once delivered.
    last_error         TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The claim index: pending notifications whose run_after has passed, oldest first.
-- FOR UPDATE SKIP LOCKED reads it under concurrency, exactly as the delivery and
-- measurement queues' ready indexes do.
CREATE INDEX report_notification_ready_idx ON report_notification (run_after)
    WHERE state = 'pending';

-- +goose Down
DROP TABLE report_notification;
