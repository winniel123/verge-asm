-- Reads and writes behind the Reports screen's recurring-reports table and its
-- "New schedule" wizard (#290, live CRUD in P0.6/T4). A report_schedule is Declared
-- and carries no timeline: a re-declaration through the wizard is a fresh insert,
-- never a recompute of an existing row (migration 21700). The row-menu's Edit is a
-- genuine in-place update of a schedule's declared contents (name / sections /
-- cadence / format / target) — a schedule carries no derived state to recompute, so
-- editing what was declared is not a recompute — and Delete is a hard delete. The
-- estate is single-tenant, so the list is unscoped; created_by attributes the admin
-- who declared each schedule and is immutable across an edit.

-- name: InsertReportSchedule :one
-- Declare one recurring report. The caller has parsed the wizard form — name, the
-- chosen sections (a JSON array), cadence, format, and the delivery target — and
-- attributes it to the admin who submitted it. sections defaults to an empty array
-- at the column, so a schedule with no sections chosen still inserts.
INSERT INTO report_schedule (name, sections, cadence, format, delivery_target, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at;

-- name: ListReportSchedules :many
-- Every declared schedule, newest-first, unbounded — the "Recurring reports" table
-- renders each row and resolves its "last delivery" from the report_delivery
-- receipts store (#291/T2), since this table holds only the declared intent.
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at
FROM report_schedule
ORDER BY id DESC;

-- name: GetReportSchedule :one
-- One declared schedule by id — the read behind the Edit wizard (prefill) and the
-- Run-now dispatch (the run reads the schedule's name/cadence/format to cut the
-- artifact for the current period). No row (pgx.ErrNoRows) is a schedule that never
-- existed or was already deleted; the caller answers a stale id rather than 500ing.
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at
FROM report_schedule
WHERE id = $1;

-- name: UpdateReportSchedule :one
-- Edit one schedule's declared contents in place (the row-menu's Edit). A schedule
-- carries no timeline and no derived state, so updating what was declared is not a
-- recompute (migration 21700) — the id, created_by and created_at are preserved.
-- Returns the updated row so the caller can confirm the target existed; no row means
-- a stale id.
UPDATE report_schedule
SET name = $2, sections = $3, cadence = $4, format = $5, delivery_target = $6
WHERE id = $1
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at;

-- name: DeleteReportSchedule :exec
-- Remove one schedule (the row-menu's Delete). A hard delete: the schedule is a
-- Declared intent, so withdrawing the declaration removes the row. Idempotent from
-- the caller's view — deleting an id already gone is not an error.
DELETE FROM report_schedule WHERE id = $1;
