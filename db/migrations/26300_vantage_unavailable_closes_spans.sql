-- +goose Up
-- ADR-2087 rules that a vantage which becomes unavailable closes EVERY open span it
-- fed, whatever the facet, and that no composition read gains an availability
-- predicate. The writer is MarkVantageUnavailable (db/queries/vantages.sql), and it
-- runs when a resolution-walk batch dead-letters. A vantage that went unavailable
-- before that writer landed is never marked again while it stays dark, so it still
-- holds every open span its last live batch left behind.
--
-- One dead prober's open `{"outcome":"reached"}` span keeps voting under ADR-0080's
-- existential Reach fold, so the exposure board reads `exposed` over a service
-- nothing can reach (#2060). This backfill is what makes the corpus agree with the
-- rule for rows written before the rule existed.
--
-- The `resolution` half does NOT repair #2077's named read: ListNameResolutionsByClass
-- reads `observation`, not `span`, so that class keeps its value until the rows age
-- past the cadence floor. What this closes is the span corpus the asset page, the
-- drift feed and the census read.
--
-- No facet is named. A facet added later is covered on the day it is added, which is
-- the deny-list move 26200 already makes for the per-vantage constraint (#2144).
--
-- This deletes nothing and rewrites no value. It draws a boundary on each open span
-- and opens a Gap behind it — the same two edges the writer draws — so ADR-0041's
-- "the Span corpus is NEVER compacted or deleted" holds. `closure_reason` stays NULL:
-- the closed union in 19000 is a withdrawal's three grounds, and an availability
-- closure is none of them. `closed_batch_id` and `opened_batch_id` stay NULL for the
-- reason 21800 already states — this is not a batch fold.
--
-- The CASE below is a spelling and not a scope: `reachability` alone decodes a lowercase
-- outcome and every other facet the capitalised one, so a facet added later takes the
-- default branch rather than a new one.
WITH closed AS (
    UPDATE span s
    SET closed_at = now()
    FROM vantage v
    WHERE v.id = s.vantage_id
      AND v.availability = 'unavailable'
      AND s.closed_at IS NULL
      AND (s.is_gap AND s.value ->> 'cause' = 'vantage-unavailable') IS NOT TRUE
    RETURNING s.subject_kind, s.subject_key, s.facet, s.discriminator,
              s.vantage_id, s.source, s.derivation
)
INSERT INTO span (
    subject_kind, subject_key, facet, discriminator, vantage_id, source,
    value, is_gap, derivation, opened_at
)
SELECT subject_kind, subject_key, facet, discriminator, vantage_id, source,
       CASE facet
           WHEN 'reachability' THEN '{"outcome":"gap","cause":"vantage-unavailable","reason":"we could not look from this position"}'::jsonb
           ELSE '{"outcome":"Gap","cause":"vantage-unavailable"}'::jsonb
       END,
       TRUE,
       -- No leaf ran, so the Gap carries the closed span's vector (ADR-0014).
       derivation,
       now()
FROM closed;

-- +goose Down
-- Irreversible by design. A Down that reopened the closed spans would have to pick
-- which of them were open before this ran, and the corpus does not record that: a
-- span closed here is indistinguishable from one the writer closed a second later.
-- ADR-0041 bars deleting the Gap rows this opened, so the Down is a no-op and the
-- boundary stays drawn.
SELECT 1;
