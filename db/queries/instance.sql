-- name: GetInstanceHealth :one
-- The instance-health tab's live database facts (#633, WORK-ORDER-DOGFOOD-R1 item 3):
-- the size of this deployment's database and its Postgres server version, read straight
-- off the running server — pg_database_size over the current database, and the
-- server_version setting. Both are Operational host facts; this touches no estate corpus.
SELECT
    pg_database_size(current_database())::bigint AS db_size_bytes,
    current_setting('server_version')::text      AS server_version;
