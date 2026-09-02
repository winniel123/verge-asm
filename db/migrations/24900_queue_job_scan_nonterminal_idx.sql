-- +goose Up
-- The hot cadence-lag gate (#1114, ADR-0137 §4) asks once per hot tick whether the
-- Scan still holds a 'ready' or 'running' job from an earlier dispatch. The answer
-- is normally NO, and a negative EXISTS has to read every candidate row, so without
-- an index the gate seq-scans the whole queue_job history on the healthy path. The
-- partial index holds only the non-terminal rows — a small set bounded by the work
-- in flight, not by the retained history.
CREATE INDEX queue_job_scan_nonterminal_idx
    ON queue_job (scan_id)
    WHERE state IN ('ready', 'running');

-- +goose Down
DROP INDEX IF EXISTS queue_job_scan_nonterminal_idx;
