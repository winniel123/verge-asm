-- Reads behind the Subjects screen (#189). All four are additive read queries
-- over the wave-0 measurement corpus (observation / batch / scan) and the seed
-- table — no new schema. `ListCurrentNameSubjects` is the thin "current Names"
-- membership read the seam note (#189 → #192) asks for: it is the one place a
-- caller decides which Names are in the estate, so a later refinement of
-- membership (Shadowed suppression, #192; the cross-class withdrawal quorum,
-- ADR-0006/ADR-0080) narrows this predicate here rather than growing a second
-- computation elsewhere.

-- name: ListCurrentNameSubjects :many
-- Every Name currently in the estate, with optional search. A Name is a member
-- while its latest resolution observation is not a measured Name Error — the
-- only route a Name leaves is our resolver measuring NameError (ADR-0006). No
-- count is selected: the estate can carry no honest denominator (ADR-0072), so
-- there is nothing here to total. A withdrawn Name (latest resolution =
-- NameError) is filtered out and reached only by key (GetNameSubject).
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM observation o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution'
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest
WHERE value->>'outcome' <> 'NameError'
  AND (@search::text = '' OR subject_key ILIKE '%' || @search::text || '%')
ORDER BY subject_key;

-- name: GetNameSubject :one
-- Resolve a Name key to at most one subject, withdrawn or not. Search is a
-- lookup and not a listing (ADR-0072 decision 3): the drill-down reaches a
-- measured-gone Name by its own key rather than manufacturing a false "no
-- record" at the URL. The caller reads the latest resolution value to decide
-- whether the subject names a population of no current member.
WITH latest AS (
    SELECT DISTINCT ON (o.subject_key)
        o.subject_key AS subject_key,
        o.value       AS value,
        o.observed_at AS observed_at
    FROM observation o
    WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = $1
    ORDER BY o.subject_key, o.observed_at DESC, o.id DESC
)
SELECT subject_key, value, observed_at
FROM latest;

-- name: GetNameCitation :one
-- The Citation chain's load-bearing hop: the observation that introduced a Name
-- — its earliest resolution observation — plus the Batch and Scan it rode in on
-- (CONTEXT.md `Citation`; ADR-0027). Answers "why is this here" by naming the
-- measurement that first admitted the subject; the chain terminates one hop
-- further at the covering Seed (FindCoveringNameSeed).
SELECT o.id, o.observed_at, o.source, o.vantage_id, o.batch_id,
       b.scan_id, sc.kind AS scan_kind
FROM observation o
JOIN batch b ON b.id = o.batch_id
JOIN scan sc ON sc.id = b.scan_id
WHERE o.subject_kind = 'name' AND o.facet = 'resolution' AND o.subject_key = $1
ORDER BY o.observed_at ASC, o.id ASC
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
