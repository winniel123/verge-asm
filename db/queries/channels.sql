-- name: CreateChannel :one
-- Returns the id only: the secret is write-only and no query hands it back.
INSERT INTO channel (url, secret, route_drift, route_coverage, route_clock, enabled, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: ListChannels :many
-- Never selects the secret: it exposes only whether one is set, so the render
-- path is structurally unable to leak it.
SELECT c.id, c.url, c.route_drift, c.route_coverage, c.route_clock, c.enabled,
       (c.secret IS NOT NULL)::boolean AS has_secret,
       c.created_by, c.created_at, c.updated_at,
       a.username AS created_by_username
FROM channel c
JOIN account a ON a.id = c.created_by
ORDER BY c.created_at DESC, c.id DESC;

-- name: GetChannel :one
-- Also omits the secret; a caller reads presence, never the value.
SELECT c.id, c.url, c.route_drift, c.route_coverage, c.route_clock, c.enabled,
       (c.secret IS NOT NULL)::boolean AS has_secret, c.created_by, c.created_at, c.updated_at
FROM channel c
WHERE c.id = $1;

-- name: UpdateChannel :exec
-- Updates everything but the secret; the secret has its own write path so an
-- edit that leaves it blank keeps the existing one untouched.
UPDATE channel
SET url = $2, route_drift = $3, route_coverage = $4, route_clock = $5,
    enabled = $6, updated_at = now()
WHERE id = $1;

-- name: SetChannelSecret :exec
-- Set, replace or clear the secret. A NULL clears it; the value is written and
-- never read back.
UPDATE channel SET secret = $2, updated_at = now() WHERE id = $1;

-- name: DeleteChannel :exec
DELETE FROM channel WHERE id = $1;
