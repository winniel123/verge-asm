-- name: CreateVantage :one
INSERT INTO vantage (host, port, username, created_by)
VALUES ($1, $2, $3, $4)
RETURNING id, host, port, username, availability, public_key, host_key, created_by, created_at;

-- name: ListVantages :many
SELECT v.id, v.host, v.port, v.username, v.availability, v.public_key, v.host_key,
       v.created_by, v.created_at, a.username AS created_by_username
FROM vantage v
JOIN account a ON a.id = v.created_by
ORDER BY v.created_at DESC, v.id DESC;

-- name: GetVantage :one
SELECT id, host, port, username, availability, public_key, host_key, created_by, created_at
FROM vantage
WHERE id = $1;

-- name: ListVantagesNeedingKey :many
-- Rows the worker still has to generate a keypair for: the public half has not
-- been published, so no key material has ever left the worker volume for them.
SELECT id, host, port, username, availability, public_key, host_key, created_by, created_at
FROM vantage
WHERE public_key IS NULL
ORDER BY id;

-- name: SetVantagePublicKey :exec
-- The worker publishes only the public half of the pair it generated on its own
-- volume; the private half never reaches Postgres.
UPDATE vantage
SET public_key = $2
WHERE id = $1;

-- name: PinVantageHostKey :exec
-- Trust-on-first-use: pin the host key only while none is pinned yet, and mark
-- the vantage available. The host_key IS NULL guard makes this a no-op once a
-- key is pinned, so a first-connect race can never overwrite an existing pin.
UPDATE vantage
SET host_key = $2, availability = 'available'
WHERE id = $1 AND host_key IS NULL;

-- name: MarkVantageUnavailable :exec
-- A pinned host key later mismatched, or the position went unreachable: the
-- vantage is marked unavailable rather than silently re-trusting a new key.
UPDATE vantage
SET availability = 'unavailable'
WHERE id = $1;
