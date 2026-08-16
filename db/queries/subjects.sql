-- Reads behind the Subjects screen (#189). All four are additive read queries
-- over the wave-0 measurement corpus (observation / batch / scan) and the seed
-- table — no new schema. `ListCurrentNameSubjects` is the thin "current Names"
-- membership read the seam note (#189 → #192) asks for: it is the one place a
-- caller decides which Names are in the estate, so a later refinement of
-- membership (Shadowed suppression, #192; the cross-class withdrawal quorum,
-- ADR-0006/ADR-0080) narrows this predicate here rather than growing a second
-- computation elsewhere.
--
-- Every derivation here reads the observation corpus through the live-tier gate
-- (#237, ADR-0041): the `cover`/`live` CTE pair below is the inlined twin of
-- ListLiveObservationsForDerivation (db/queries/retention.sql), evaluated against
-- the caller's read instant @as_of with k = @floor_cadences, so an evidential row
-- (past its own per-timeline bound, or on a timeline no enabled Scan covers) is
-- structurally unreadable here the instant it crosses that bound — never merely
-- absent after the Retirer's next sweep. The gate cannot be a parameterless VIEW
-- because it carries the read instant, so it is inlined at each read.

-- name: ListCurrentNameSubjects :many
-- Every Name currently in the estate, with optional search. A Name is a member
-- while its latest resolution observation neither reads a measured Name Error nor
-- is Shadowed: resolution-walk's NameError (the name does not exist) and
-- wildcard-discrimination's Shadowed (a wildcard-synthesised answer) both suppress
-- a Name's membership as affirmatively as each other (#192; ADR-0006, ADR-0086).
-- No count is selected: the estate can carry no honest denominator (ADR-0072), so
-- there is nothing here to total. A suppressed Name is filtered out and reached
-- only by key (GetNameSubject). Reads through the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE value->>'outcome' NOT IN ('NameError', 'Shadowed')
  AND (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetNameSubject :one
-- Resolve a Name key to at most one subject, withdrawn or not. Search is a
-- lookup and not a listing (ADR-0072 decision 3): the drill-down reaches a
-- measured-gone Name by its own key rather than manufacturing a false "no
-- record" at the URL. The caller reads the latest resolution value to decide
-- whether the subject names a population of no current member. Reads through the
-- live-tier gate (#237): a Name is measured-gone by VALUE (a NameError/Shadowed
-- latest), which is a live measurement — the gate removes only rows aged past
-- their own bound, so a currently-measured subject is always reachable here.
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: GetNameCitation :one
-- The Citation chain's load-bearing hop: the observation that introduced a Name
-- — its earliest LIVE resolution observation — plus the Batch and Scan it rode in
-- on (CONTEXT.md `Citation`; ADR-0027). Answers "why is this here" by naming the
-- measurement that admitted the subject; the chain terminates one hop further at
-- the covering Seed (FindCoveringNameSeed). Reads through the live-tier gate
-- (#237): the introducing observation is the earliest one still within the live
-- tier, so the chain rests on a measurement a derivation may still read, not on an
-- evidential row.
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
)
SELECT o.id, o.observed_at, o.source, o.vantage_id, o.batch_id,
       b.scan_id, sc.kind AS scan_kind
FROM live o
JOIN batch b ON b.id = o.batch_id
JOIN scan sc ON sc.id = b.scan_id
WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = @subject_key
ORDER BY o.observed_at ASC, o.id ASC
LIMIT 1;

-- name: ListCurrentServiceSubjects :many
-- Every Service currently in the estate, with optional search (#195). A Service
-- is an (Address, port, transport) triple whose membership is its Address's
-- membership restated — an Address is in the estate exactly while a current
-- resolution cites it or a Seed covers it — so this is the thin "current
-- Services" read the drill-down lists. Like the Name listing it carries no
-- denominator (ADR-0072). A Service that has fallen out of the estate (its
-- Address de-cited) is reached only by its own key; the value shown is the latest
-- reachability verdict, reached or not-reached, both measured values. Reads
-- through the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'service' AND o.facet = 'reachability'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetServiceSubject :one
-- Resolve a Service key to at most one subject (#195). A Service drill-down
-- reaches a subject by its own key — including one whose Address has left the
-- estate, which is not a false "no record" but a population of no current member
-- (ADR-0072). The caller reads the latest reachability value to render the
-- current verdict and the Address the triple sits on. Reads through the live-tier
-- gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'service' AND o.facet = 'reachability' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: ListCurrentEndpointSubjects :many
-- Every Endpoint currently in the estate, with optional search (#198). An Endpoint
-- is a (Name, Service) pair — keyed `name@service`, or `@service` for the nameless
-- endpoint — the only key under which HTTP identity is single-valued (CONTEXT.md
-- `Endpoint`). Its membership rides its Service's (the Address's membership
-- restated), so this is the thin "current Endpoints" read the drill-down lists.
-- Like the Name and Service listings it carries no denominator (ADR-0072). The
-- value shown is the latest http-identity the http-exchange leaf recorded. Reads
-- through the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'endpoint' AND o.facet = 'http-identity'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetEndpointSubject :one
-- Resolve an Endpoint key to at most one subject (#198). An Endpoint drill-down
-- reaches a subject by its own key — including one whose Service has left the
-- estate, which is a population of no current member rather than a false "no
-- record" (ADR-0072). The caller reads the latest http-identity value to render
-- the current HTTP identity and split the key into its Name and Service legs.
-- Reads through the live-tier gate (#237).
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.subject_kind = 'endpoint' AND o.facet = 'http-identity' AND o.subject_key = @subject_key
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: FindNameCitingAddress :one
-- A Name whose current resolution cites the given Address (#195) — the Citation
-- hop that answers why a Service's Address is in the estate. An Address has no
-- lifecycle of its own, so its membership is grounded in evidence about ANOTHER
-- subject: the Name whose Resolved answer names it. Where a resolution stops
-- citing the Address this returns no row, which is exactly the `uncited` ground a
-- departure records. Best-effort: the longest-lived citing Name, one hop. Reads
-- through the live-tier gate (#237): the citing resolution must be one a
-- derivation may still read, so a Name held only by an evidential answer no longer
-- keeps an Address in the estate.
WITH cover AS (
    SELECT o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source,
           MIN(s.cadence_seconds) AS tightest_cadence
    FROM observation o
    JOIN batch b ON b.id = o.batch_id
    JOIN scan  s ON s.id = b.scan_id AND s.enabled = TRUE
    GROUP BY o.subject_key, o.facet, o.discriminator, o.vantage_id, o.source
),
live AS (
    SELECT o.id, o.facet, o.subject_kind, o.subject_key, o.discriminator,
           o.vantage_id, o.source, o.value, o.observed_at, o.batch_id
    FROM observation o
    JOIN cover c
        ON  c.subject_key   = o.subject_key
        AND c.facet         = o.facet
        AND c.discriminator = o.discriminator
        AND c.vantage_id IS NOT DISTINCT FROM o.vantage_id
        AND c.source        = o.source
    WHERE EXTRACT(EPOCH FROM (sqlc.arg(as_of)::timestamptz - o.observed_at))
          <= sqlc.arg(floor_cadences)::bigint * c.tightest_cadence
),
latest AS (
    SELECT DISTINCT ON (o.subject_key, o.vantage_id)
        o.subject_key AS subject_key,
        o.value->>'outcome' AS outcome,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM live o
    WHERE o.facet = 'resolution' AND o.subject_kind = 'name'
    ORDER BY o.subject_key, o.vantage_id, o.observed_at DESC
)
SELECT subject_key, observed_at
FROM latest
WHERE outcome = 'Resolved'
  AND value->'addresses' ? @address::text
ORDER BY observed_at ASC, subject_key ASC
LIMIT 1;

-- name: FindCoveringAddressSeed :one
-- The address-scope Seed a Service's Address falls inside (#195) — the other
-- limb of Address membership, and where the Citation chain terminates when no
-- resolution cites the Address. Native CIDR containment (`>>=`) is a test over
-- the address and never its spelling, so the gate cannot turn on a rendering
-- (CONTEXT.md `Seed`). The most specific covering scope wins where scopes nest.
SELECT s.id, s.address_cidr, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
WHERE s.kind = 'address' AND s.address_cidr IS NOT NULL
  AND s.address_cidr >>= @address::inet
ORDER BY masklen(s.address_cidr) DESC
LIMIT 1;

-- name: FindCoveringNameSeed :one
-- The Seed a Name's Citation chain terminates at: the name scope whose query set
-- the dns Scan was drawn from (CONTEXT.md `Citation` — every chain bottoms out at
-- a Seed or a declared source). Wave-0 measures the seed domains themselves; the
-- label-wise suffix match also carries a later enumerated subdomain to its scope,
-- and the longest matching domain wins when scopes nest.
SELECT s.id, s.name_domain, s.created_at, a.username AS created_by_username
FROM seed s
JOIN account a ON a.id = s.created_by
WHERE s.kind = 'name' AND s.name_domain IS NOT NULL
  AND (@name::text = s.name_domain OR @name::text LIKE '%.' || s.name_domain)
ORDER BY length(s.name_domain) DESC
LIMIT 1;
