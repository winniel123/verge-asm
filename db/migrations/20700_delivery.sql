-- +goose Up
-- A Delivery is the Operational record of one outbound POST of one Message to one
-- Channel (v1 spec §4.5, notification-channels.md §8, ADR-0039). It has NO cause:
-- a delivery failure is not the world moving, our looking changing, or a clock
-- crossing, so it never becomes a Message and never touches Coverage. It travels
-- with its Message and holds no retention rule of its own — the Message corpus is
-- what the operator reads back, and a Message renders its own delivery outcomes
-- from this table (#139/ADR-0081).
--
-- The record IS the "undelivered mark": a dead-lettered delivery is a row in
-- state 'undelivered'. The Message row is never touched — it holds read-state and
-- no other operator state (CONTEXT.md `Message`), so a dead-lettered delivery
-- neither deletes nor hides the Message; it is simply visible as an undelivered
-- delivery joined to it. A dead-lettered Delivery licenses no silence.
--
-- Retry runs on the queue's own retry/backoff/dead-letter machinery (#188): five
-- attempts over roughly an hour, exponential, then dead-lettered. The state and
-- attempt columns mirror queue_job; the Message identifier is the receiver's
-- de-duplication key and is stable across retries, so a retry advances THIS row's
-- attempt rather than minting a second delivery — the unique (message, channel)
-- key enforces one delivery per routed pair.
CREATE TABLE delivery (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    message_id    BIGINT NOT NULL REFERENCES message (id),
    channel_id    BIGINT NOT NULL REFERENCES channel (id),
    -- pending: attempts remain and run_after gates the next try.
    -- sending:  claimed by a worker, in flight.
    -- delivered: a 2xx was received.
    -- undelivered: dead-lettered after the attempt budget — the undelivered mark.
    state         TEXT NOT NULL DEFAULT 'pending'
                  CHECK (state IN ('pending', 'sending', 'delivered', 'undelivered')),
    attempt       INT NOT NULL DEFAULT 1,
    max_attempts  INT NOT NULL DEFAULT 5,
    -- The last error string, shown as channel-surface drill-down only (#22): raw
    -- errors are never a top-level log. NULL once delivered.
    last_error    TEXT,
    run_after     TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT delivery_message_channel UNIQUE (message_id, channel_id)
);

-- The claim index: pending deliveries whose run_after has passed, oldest first.
-- FOR UPDATE SKIP LOCKED reads it under concurrency, exactly as the measurement
-- queue's ready index does.
CREATE INDEX delivery_ready_idx ON delivery (run_after) WHERE state = 'pending';

-- +goose Down
DROP TABLE delivery;
