-- +goose Up
-- Dispatch retention (v1 spec §4.6, ADR-0041): `Dispatch` is the one corpus a
-- wall clock may retire — it carries no observations and the comparison path is
-- structurally barred from reading it. Retiring an expired Dispatch must never
-- reach the Observation or Span corpus, so the operational back-references from
-- `batch` and `queue_job` are moved to ON DELETE SET NULL: deleting a Dispatch
-- severs the operational provenance pointer and nothing more. A Batch keeps its
-- own `scan_id` and every Observation it produced; a `queue_job` keeps its
-- `batch_id`. The delete therefore cannot cascade into measured data — the
-- structural half of "a wall clock may retire Dispatch and only Dispatch".
--
-- #206 seeded `retention_settings.dispatch_cadence_multiple` (a multiple of the
-- slowest enabled Scan's cadence, not a day count) defaulted to 0 == unbounded,
-- with the floor deferred here. This migration lands only the FK change; the
-- floor and the sweep are code (cmd/web/settings.go, internal/retention).
ALTER TABLE batch DROP CONSTRAINT batch_dispatch_id_fkey;
ALTER TABLE batch ADD CONSTRAINT batch_dispatch_id_fkey
    FOREIGN KEY (dispatch_id) REFERENCES dispatch (id) ON DELETE SET NULL;

ALTER TABLE queue_job DROP CONSTRAINT queue_job_dispatch_id_fkey;
ALTER TABLE queue_job ADD CONSTRAINT queue_job_dispatch_id_fkey
    FOREIGN KEY (dispatch_id) REFERENCES dispatch (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE queue_job DROP CONSTRAINT queue_job_dispatch_id_fkey;
ALTER TABLE queue_job ADD CONSTRAINT queue_job_dispatch_id_fkey
    FOREIGN KEY (dispatch_id) REFERENCES dispatch (id);

ALTER TABLE batch DROP CONSTRAINT batch_dispatch_id_fkey;
ALTER TABLE batch ADD CONSTRAINT batch_dispatch_id_fkey
    FOREIGN KEY (dispatch_id) REFERENCES dispatch (id);
