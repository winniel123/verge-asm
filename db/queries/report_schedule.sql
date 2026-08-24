-- Reads and writes behind the Reports screen's recurring-reports table and its
-- "New schedule" wizard (#290, live CRUD in P0.6/T4). A report_schedule is Declared
-- and carries no timeline: a re-declaration through the wizard is a fresh insert,
-- never a recompute of an existing row (migration 21700). The row-menu's Edit is a
-- genuine in-place update of a schedule's declared contents (name / sections /
-- cadence / format / channel) — a schedule carries no derived state to recompute, so
-- editing what was declared is not a recompute — and Delete is a hard delete. The
-- estate is single-tenant, so the list is unscoped; created_by attributes the admin
-- who declared each schedule and is immutable across an edit.
--
-- The schedule's delivery destination is a Channel: channel_id binds the signed-HTTPS
-- Channel that receives the run's link-only ready-message, and NULL is download-only
-- (P0.6c/T7, #508, migration 22700). The free-text delivery_target is superseded by
-- the binding — it is written empty and no longer read as the destination.

-- name: InsertReportSchedule :one
-- Declare one recurring report. The caller has parsed the wizard form — name, the
-- chosen sections (a JSON array), cadence, format, and the delivery destination
-- (a channel_id, or NULL for download-only) — and attributes it to the admin who
-- submitted it. sections defaults to an empty array at the column, so a schedule with
-- no sections chosen still inserts. delivery_target is written empty (superseded by
-- the channel binding).
INSERT INTO report_schedule (name, sections, cadence, format, delivery_target, channel_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id;

-- name: ListReportSchedules :many
-- Every declared schedule, newest-first, unbounded — the "Recurring reports" table
-- renders each row (resolving the bound channel's URL for the Delivery cell) and its
-- "last delivery" from the report_delivery receipts store (#291/T2), since this table
-- holds only the declared intent.
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id
FROM report_schedule
ORDER BY id DESC;

-- name: GetReportSchedule :one
-- One declared schedule by id — the read behind the Edit wizard (prefill, including
-- the bound channel) and the Run-now dispatch (the run reads the schedule's
-- name/cadence/format to cut the artifact for the current period). No row
-- (pgx.ErrNoRows) is a schedule that never existed or was already deleted; the caller
-- answers a stale id rather than 500ing.
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id
FROM report_schedule
WHERE id = $1;

-- name: UpdateReportSchedule :one
-- Edit one schedule's declared contents in place (the row-menu's Edit). A schedule
-- carries no timeline and no derived state, so updating what was declared is not a
-- recompute (migration 21700) — the id, created_by and created_at are preserved.
-- channel_id is part of the declared contents, so an edit can rebind the destination
-- or set it to download-only (NULL). Returns the updated row so the caller can confirm
-- the target existed; no row means a stale id.
UPDATE report_schedule
SET name = $2, sections = $3, cadence = $4, format = $5, delivery_target = $6, channel_id = $7
WHERE id = $1
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id;

-- name: DeleteReportSchedule :exec
-- Remove one schedule (the row-menu's Delete). A hard delete: the schedule is a
-- Declared intent, so withdrawing the declaration removes the row. Idempotent from
-- the caller's view — deleting an id already gone is not an error.
DELETE FROM report_schedule WHERE id = $1;
