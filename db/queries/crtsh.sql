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

-- name: ListAdmittedNames :many
-- The distinct CT-admitted names the dns Scan also resolves (ADR-0107, wave-1).
-- A source that admits without observing leaves an admitted_name row per Name it
-- named; unioned into the dns Scan's resolution set, each acquires a resolution
-- timeline from our own resolver and becomes a measured member or leaves by Name
-- Error (ADR-0027, ADR-0096 §1). DISTINCT because an append-only source re-admits
-- the same names on every poll; unconditional of the source's current enablement,
-- since resolution is the dns Scan's act and a Name leaves only by measurement.
SELECT DISTINCT name
FROM admitted_name
ORDER BY name;

-- name: CTLastBatchAdmitCount :one
-- How many Names the most recent bulk `ct` Batch admitted (#880, spec §6.2). The
-- active-source hero's run readout states "last ct scan · <source> · <age> · <n> names
-- admitted"; this is that <n>. It counts the admitted_name rows citing the newest
-- kind='ct' Batch (the last bulk run, whichever source produced it — the drift tail's
-- kind='ct-tail' Batches are excluded). A dead-lettered or empty run admits nothing, so
-- 0 is a truthful count, and COALESCE gives 0 when no ct Batch has ever run. One scalar
-- row always returns.
SELECT COALESCE((
    SELECT count(*)
    FROM admitted_name an
    WHERE an.batch_id = (
        SELECT b.id FROM batch b
        WHERE b.kind = 'ct'
        ORDER BY b.created_at DESC, b.id DESC
        LIMIT 1
    )
), 0)::bigint AS names;

-- name: ReserveCTSlot :one
-- Atomically claim the next free slot for one CT fetch of a given source,
-- instance-wide (ADR-0005: the throttle is per-source across the whole instance,
-- in Postgres, not worker memory). GREATEST(next_free_at, now()) is this request's
-- slot; next_free_at advances one interval past it, so concurrent workers each
-- reserve a distinct, correctly-spaced slot. The interval is the source's own
-- spacing, supplied by the caller. The caller waits until slot_at before going on
-- the wire.
WITH reserved AS (
    UPDATE ct_throttle
    SET next_free_at = GREATEST(next_free_at, now())
        + make_interval(secs => sqlc.arg(interval_seconds)::double precision)
    WHERE source = sqlc.arg(source)
    RETURNING next_free_at
)
SELECT (next_free_at
    - make_interval(secs => sqlc.arg(interval_seconds)::double precision))::timestamptz AS slot_at
FROM reserved;
