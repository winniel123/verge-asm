-- Reads and writes behind the Reports screen's recurring-reports table and its
-- "New schedule" wizard (#290). A report_schedule is Declared and carries no
-- timeline: there is a plain insert and an unbounded newest-first list, no
-- content update and no delete (the row-menu's edit/delete stay out of scope until
-- the scheduling dispatcher lands). The estate is single-tenant, so the list is
-- unscoped; created_by attributes the admin who declared each schedule.

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
-- renders each row and resolves its "last delivery" from the Message corpus, since
-- deliveries are messages (ADR-0039, ADR-0081) and this table holds only intent.
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at
FROM report_schedule
ORDER BY id DESC;
