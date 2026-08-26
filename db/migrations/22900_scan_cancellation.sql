-- +goose Up
-- Stop-dispatch / terminate (DF-F4, package v3.16.2). An admin may end a Dispatch
-- that is in flight — either a graceful STOP (pending jobs cancelled, running jobs
-- allowed to finish and commit) or a hard TERMINATE (running jobs cancelled too,
-- their uncommitted work discarded). Both are Operational acts on the queue corpus;
-- the drift engine never reads dispatch or queue_job (ADR-0041), so recording a
-- terminal disposition here moves no estate truth.
--
-- The Dispatch gains two terminal statuses beside 'fanned-out': 'stopped' (recorded
-- stopped · partial — some jobs ran, some were cancelled) and 'terminated' (recorded
-- terminated — the run was killed). A natural completion leaves the status
-- 'fanned-out'; these mark only an operator-ended run.
ALTER TABLE dispatch DROP CONSTRAINT IF EXISTS dispatch_status_check;
ALTER TABLE dispatch
    ADD CONSTRAINT dispatch_status_check
    CHECK (status IN ('fanned-out', 'stopped', 'terminated'));

-- A queue job gains a 'cancelled' terminal state: a pending (ready) job a stop or
-- terminate cancelled before the worker claimed it, or a running job a terminate
-- cancelled out from under the worker. 'cancelled' is terminal and never claimable —
-- ClaimJob selects state = 'ready' alone, so a cancelled job leaves the ready set the
-- instant it is marked. It carries no batch: a cancelled job produced no committed
-- observations (a running job whose work already committed is 'done'/'dead', its
-- batch append-only and untouched).
ALTER TABLE queue_job DROP CONSTRAINT IF EXISTS queue_job_state_check;
ALTER TABLE queue_job
    ADD CONSTRAINT queue_job_state_check
    CHECK (state IN ('ready', 'running', 'done', 'dead', 'retried', 'cancelled'));

-- +goose Down
ALTER TABLE queue_job DROP CONSTRAINT IF EXISTS queue_job_state_check;
ALTER TABLE queue_job
    ADD CONSTRAINT queue_job_state_check
    CHECK (state IN ('ready', 'running', 'done', 'dead', 'retried'));
ALTER TABLE dispatch DROP CONSTRAINT IF EXISTS dispatch_status_check;
ALTER TABLE dispatch
    ADD CONSTRAINT dispatch_status_check
    CHECK (status IN ('fanned-out'));
