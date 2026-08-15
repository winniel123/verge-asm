-- name: RecordHeartbeat :one
INSERT INTO heartbeat DEFAULT VALUES
RETURNING id, checked_at;
