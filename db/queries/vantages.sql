-- name: CreateVantage :one
-- Provisioning a prober creates a Vantage with connection detail. Its
-- measurement identity is still mandatory: the caller derives `name` from the
-- endpoint (username@host:port) so it is unique per provisioned endpoint, class
-- defaults to 'unverified' until a prober re-verifies it, and resolver ships
-- blank ('') for the operator to set. availability starts 'pending' — no host
-- key has been pinned yet. The explicit casts keep the params plain scalars even
-- though the prober columns are nullable on the table.
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
          public_key, host_key, created_by, created_at, latency_ms;

-- name: ListVantages :many
-- The web prober list: only provisioned vantages (those carrying a prober
-- endpoint). The resolver-only `local` vantage has no prober and is excluded.
-- latency_ms is the per-vantage connect round-trip the Dashboard renders (P0.5),
-- NULL until the prober connect that pins the host key lands a first measurement.
SELECT v.id, v.name, v.class, v.resolver, v.host, v.port, v.username,
       v.availability, v.public_key, v.host_key, v.created_by, v.created_at,
       v.latency_ms, a.username AS created_by_username
FROM vantage v
JOIN account a ON a.id = v.created_by
WHERE v.host IS NOT NULL
ORDER BY v.created_at DESC, v.id DESC;

-- name: GetVantage :one
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms
FROM vantage
WHERE id = $1;

-- name: ListUnavailableVantages :many
-- The Coverage register of positions we currently cannot observe from
-- (ADR-0108). It includes the resolver-only `local` vantage — which ListVantages
-- excludes for the prober list — because that is exactly the position whose
-- resolver going unreachable this surface must make loud. Ordered by name so the
-- rendering is stable.
SELECT id, name, class, resolver, availability
FROM vantage
WHERE availability = 'unavailable'
ORDER BY name;

-- name: ListVantagesNeedingKey :many
-- Rows the worker still has to generate a keypair for: a provisioned prober
-- (host set) whose public half has not been published, so no key material has
-- ever left the worker volume for them.
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms
FROM vantage
WHERE host IS NOT NULL AND public_key IS NULL
ORDER BY id;

-- name: ListVantagesNeedingLatency :many
-- Rows the worker still has to measure a connect latency for (P0.5): a
-- provisioned prober (host set) whose keypair has been published (public_key set,
-- so a private half exists on the worker volume to dial with) but whose latency
-- has never been measured. The connect the worker makes here is the same one that
-- pins the host key trust-on-first-use, so measuring on it needs no extra dial.
SELECT id, name, class, resolver, host, port, username, availability,
       public_key, host_key, created_by, created_at, latency_ms
FROM vantage
WHERE host IS NOT NULL AND public_key IS NOT NULL AND latency_ms IS NULL
ORDER BY id;

-- name: SetVantageLatency :exec
-- The worker records the round-trip time of the prober connect that pinned the
-- host key (P0.5, SPEC-CHANGE.md collision #7). Stored in whole milliseconds — the
-- unit the Dashboard renders — and set only from a real measurement, never a
-- fabricated value.
UPDATE vantage
SET latency_ms = $2
WHERE id = $1;

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

-- name: MarkVantageAvailable :exec
-- A completed Batch at this vantage is proof the position can observe again, so
-- Availability is derived back to 'available' from the terminal batch outcome
-- (ADR-0108). A host-key-mismatched prober cannot complete a Batch — its SSH
-- connection is refused before any measurement runs — so this can never silently
-- clear the trust-on-first-use pin MarkVantageUnavailable set.
UPDATE vantage
SET availability = 'available'
WHERE id = $1;
