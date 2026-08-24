-- +goose Up
-- A report_delivery is the Operational receipt of one run of a report_schedule
-- (#291/T2): the record that a scheduled report was cut for a bounded period and,
-- where it left, delivered. It is Operational and carries no cause — a report run
-- is neither the world moving, our looking changing, nor a clock crossing, so it
-- never becomes a Message and never touches the comparison path (ADR-0039,
-- ADR-0081). It backs the "Recurring reports" table's "last sent" cell and the
-- delivered-artifact view, replacing the older read that resolved "last delivery"
-- from the Message corpus (report_schedule holds only the declared intent).
--
-- The receipt is deliberately thin: it stores the period bounds the run covered and
-- its outcome, NOT a snapshot of the estate rows the artifact renders. The delivered
-- document recomputes from period_start/period_end at render time, so a receipt
-- never freezes estate state and never carries the estate off-instance — it is an
-- operational record of a run, nothing more (ADR-0039).
--
-- This table does NOT reference message or channel: a report run is neither a
-- Message nor a channel POST, so it is not modelled on the delivery table (which
-- keys on message_id/channel_id). It stands on its own, keyed to the schedule it
-- ran, with a per-schedule 1-based sequence so a run reads as "delivery #N".
CREATE TABLE report_delivery (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    schedule_id   BIGINT NOT NULL REFERENCES report_schedule (id),
    -- The window the run covered, half-open by convention. The artifact recomputes
    -- its contents from these bounds at render time; the receipt snapshots nothing.
    period_start  TIMESTAMPTZ NOT NULL,
    period_end    TIMESTAMPTZ NOT NULL,
    -- The per-schedule 1-based sequence number of this run, rendered as "delivery #N"
    -- on the artifact. Assigned max+1 at insert; the unique key below keeps it dense.
    delivery_no   INT NOT NULL,
    -- When the artifact was cut. delivered_at stamps when it actually left, and is
    -- NULL where the run was generated but not (yet) delivered — a download-only
    -- schedule generates without delivering, so a receipt can stand without it.
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at  TIMESTAMPTZ,
    -- generated: the artifact was cut; no outbound delivery was attempted or it is
    --            pending. delivered: the artifact left to its target. failed: the
    --            run did not complete — not a delivery to view.
    state         TEXT NOT NULL
                  CHECK (state IN ('generated', 'delivered', 'failed')),

    CONSTRAINT report_delivery_schedule_no UNIQUE (schedule_id, delivery_no)
);

-- "Latest per schedule": the recurring-reports table resolves each row's last
-- delivery, and the artifact view opens the newest, by (schedule_id, id DESC).
CREATE INDEX report_delivery_latest ON report_delivery (schedule_id, id DESC);

-- +goose Down
DROP TABLE report_delivery;
