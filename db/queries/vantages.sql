-- name: CreateVantage :one
INSERT INTO vantage (name, host, port, username, availability, created_by)
VALUES (
    sqlc.arg(name)::text,
    sqlc.arg(host)::text,
    sqlc.arg(port)::int,
    sqlc.arg(username)::text,
    'pending',
    sqlc.arg(created_by)::bigint
)
RETURNING id, name, class, resolver, host, port, username, availability,
          public_key, host_key, created_by, created_at, latency_ms, platform, egress,
          dialled_addr;

-- name: ListVantages :many
SELECT v.id, v.name, v.class, v.resolver, v.host, v.port, v.username,
       v.availability, v.public_key, v.host_key, v.created_by, v.created_at,
       v.latency_ms, v.platform, v.egress, v.dialled_addr,
       a.username AS created_by_username
FROM vantage v
JOIN account a ON a.id = v.created_by
WHERE v.host IS NOT NULL
ORDER BY v.created_at DESC, v.id DESC;

-- name: GetVantage :one
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE id = $1;

-- name: ListUnavailableVantages :many
SELECT id, name, class, resolver, availability
FROM vantage
WHERE availability = 'unavailable'
ORDER BY name;

-- name: ListVantagesNeedingKey :many
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE host IS NOT NULL AND public_key IS NULL
ORDER BY id;

-- name: ListVantagesNeedingLatency :many
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms, platform, egress,
       dialled_addr
FROM vantage
WHERE host IS NOT NULL AND public_key IS NOT NULL AND latency_ms IS NULL
ORDER BY id;

-- name: SetVantageLatency :exec
UPDATE vantage
SET latency_ms = $2
WHERE id = $1;

-- name: SetVantageProbeFacts :exec
UPDATE vantage
SET platform = $2, egress = $3, dialled_addr = $4
WHERE id = $1;

-- name: SetVantagePublicKey :exec
UPDATE vantage
SET public_key = $2
WHERE id = $1;

-- name: PinVantageHostKey :exec
UPDATE vantage
SET host_key = $2, availability = 'available'
WHERE id = $1 AND host_key IS NULL;

-- name: MarkVantageUnavailable :exec
UPDATE vantage
SET availability = 'unavailable'
WHERE id = $1;

-- name: MarkVantageAvailable :exec
UPDATE vantage
SET availability = 'available'
WHERE id = $1;
