-- name: GetInstanceHealth :one
SELECT
    pg_database_size(current_database())::bigint AS db_size_bytes,
    current_setting('server_version')::text      AS server_version;
