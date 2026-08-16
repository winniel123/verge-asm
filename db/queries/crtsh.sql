-- name: ListNameSeeds :many
-- The name-scope Seeds the CT Scan queries — id and registrable domain, one
-- crt.sh query per row (ADR-0106). Distinct from ListNameSeedDomains (domains
-- only, for the dns Scan): a CT admission's Citation chain terminates at the Seed,
-- so its id travels with the domain (ADR-0027).
SELECT id, name_domain
FROM seed
WHERE kind = 'name' AND name_domain IS NOT NULL
ORDER BY id;

-- name: InsertAdmittedName :exec
-- One CT admission (ADR-0027, ADR-0106): a Name a crt.sh Batch admitted, carrying
-- the Batch that admitted it (the Citation hop) and the covering name-scope Seed
-- the chain terminates at. No observation, no facet, no timeline — admission is
-- not membership (ADR-0096 §5).
INSERT INTO admitted_name (name, source, seed_id, batch_id)
VALUES ($1, $2, $3, $4);

-- name: ReserveCTSlot :one
-- Atomically claim the next free slot for one crt.sh fetch, instance-wide
-- (ADR-0005: the 5 req/min throttle is per-source across the whole instance, in
-- Postgres, not worker memory). GREATEST(next_free_at, now()) is this request's
-- slot; next_free_at advances one interval past it, so concurrent workers each
-- reserve a distinct, correctly-spaced slot. The caller waits until slot_at
-- before going on the wire.
WITH reserved AS (
    UPDATE crtsh_throttle
    SET next_free_at = GREATEST(next_free_at, now())
        + make_interval(secs => sqlc.arg(interval_seconds)::double precision)
    WHERE id = 1
    RETURNING next_free_at
)
SELECT (next_free_at
    - make_interval(secs => sqlc.arg(interval_seconds)::double precision))::timestamptz AS slot_at
FROM reserved;
