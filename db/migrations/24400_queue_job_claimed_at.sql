-- +goose Up
-- The stale-`running` reaper's lease clock (#853). A job stuck in state 'running'
-- because the worker process died or hung mid-job is never reclaimed: ClaimJob
-- claims state 'ready' alone, and the only exits from 'running' are the owning
-- worker's guarded terminal writes or an operator Terminate. That orphans the job
-- and wedges its Dispatch as "in flight" forever. The reaper (internal/queue/
-- reaper.go) needs to know WHEN a job entered 'running' to decide it is stale, so
-- ClaimJob now stamps claimed_at at the instant it marks the winner 'running'.
--
-- claimed_at is the claim instant, distinct from run_after (the schedule gate) and
-- created_at (the enqueue instant). It is NULL for a job that has never been
-- claimed — a 'ready', 'cancelled' or 'retried' row. The reaper reads it only for
-- 'running' rows, and a NULL never satisfies the `< cutoff` test, so a row with no
-- lease is never reaped.
ALTER TABLE queue_job ADD COLUMN claimed_at TIMESTAMPTZ;

-- Backfill any job already 'running' at deploy: stamp its lease now so the reaper
-- gives it a full threshold from this instant before it is eligible, rather than
-- reaping it immediately on a NULL that the `< cutoff` test would skip anyway. This
-- keeps a genuinely-in-flight job at deploy safe and starts its clock honestly.
UPDATE queue_job SET claimed_at = now() WHERE state = 'running' AND claimed_at IS NULL;

-- The reaper's sweep index: the stale-running scan reads 'running' rows by their
-- lease instant. A partial index keyed on claimed_at, scoped to state = 'running',
-- keeps the sweep off the full table the way queue_job_ready_idx keeps ClaimJob off
-- it — the running set is small, but the index keeps the periodic sweep O(stale)
-- rather than O(all jobs).
CREATE INDEX queue_job_running_idx ON queue_job (claimed_at) WHERE state = 'running';

-- +goose Down
DROP INDEX IF EXISTS queue_job_running_idx;
ALTER TABLE queue_job DROP COLUMN claimed_at;
