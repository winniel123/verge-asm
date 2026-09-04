-- name: CreateChannel :one
INSERT INTO channel (url, secret, route_drift, route_coverage, route_clock, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: ListChannels :many
SELECT c.id, c.url, c.route_drift, c.route_coverage, c.route_clock, c.enabled,
       (c.secret IS NOT NULL)::boolean AS has_secret,
       c.created_by, c.created_at, c.updated_at,
       a.username AS created_by_username
FROM channel c
JOIN account a ON a.id = c.created_by
ORDER BY c.created_at DESC, c.id DESC;

-- name: GetChannel :one
SELECT c.id, c.url, c.route_drift, c.route_coverage, c.route_clock, c.enabled,
       (c.secret IS NOT NULL)::boolean AS has_secret, c.created_by, c.created_at, c.updated_at
FROM channel c
WHERE c.id = $1;

-- name: UpdateChannel :exec
UPDATE channel
SET url = $2, route_drift = $3, route_coverage = $4, route_clock = $5,
    enabled = $6, updated_at = now()
WHERE id = $1;

-- name: SetChannelSecret :exec
UPDATE channel SET secret = $2, updated_at = now() WHERE id = $1;

-- name: DeleteChannel :exec
DELETE FROM channel WHERE id = $1;
