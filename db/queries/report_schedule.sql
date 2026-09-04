-- name: InsertReportSchedule :one
INSERT INTO report_schedule (name, sections, cadence, format, delivery_target, channel_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id;

-- name: ListReportSchedules :many
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id
FROM report_schedule
ORDER BY id DESC;

-- name: GetReportSchedule :one
SELECT id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id
FROM report_schedule
WHERE id = $1;

-- name: UpdateReportSchedule :one
UPDATE report_schedule
SET name = $2, sections = $3, cadence = $4, format = $5, delivery_target = $6, channel_id = $7
WHERE id = $1
RETURNING id, name, sections, cadence, format, delivery_target, created_by, created_at, channel_id;

-- name: DeleteReportSchedule :exec
DELETE FROM report_schedule WHERE id = $1;
