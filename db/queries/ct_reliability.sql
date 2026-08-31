-- name: InsertCTReliabilitySample :exec
-- Record one bulk-by-name query as a reliability sample (spec §3, #879): the source
-- it ran against, whether it succeeded (a well-formed 200), its end-to-end fetch
-- latency in whole milliseconds, and whether a successful query returned zero
-- certificate names (the false-empty limb). One row per query attempt, so a retry is
-- its own sample.
INSERT INTO ct_reliability_sample (source, ok, latency_ms, empty)
VALUES ($1, $2, $3, $4);

-- name: TrimCTReliabilitySamples :exec
-- Keep only the newest `keep` samples for one source, so the table stays bounded and
-- the bar is measured over a rolling window rather than all history (spec §3). Run
-- after each insert. Ordered newest-first, id breaking an observed_at tie, matching
-- the read window's order.
DELETE FROM ct_reliability_sample AS s
WHERE s.source = sqlc.arg(source)
  AND s.id NOT IN (
      SELECT keep.id FROM ct_reliability_sample AS keep
      WHERE keep.source = sqlc.arg(source)
      ORDER BY keep.observed_at DESC, keep.id DESC
      LIMIT sqlc.arg(keep_count)
  );

-- name: CTReliabilityWindow :one
-- Aggregate one source's newest `window` samples into the three bar limbs (spec §3):
-- the total measured, how many succeeded, how many succeeded but returned zero names
-- (false-empty), and the p95 end-to-end latency over the window. percentile_disc
-- returns an actual sampled latency, and COALESCE gives 0 for an empty window. The
-- caller (internal/scan.EvaluateCTReliability) turns these into pass/fail per limb.
WITH w AS (
    SELECT ok, latency_ms, empty
    FROM ct_reliability_sample
    WHERE source = sqlc.arg(source)
    ORDER BY observed_at DESC, id DESC
    LIMIT sqlc.arg(sample_size)
)
SELECT
    count(*)::bigint AS total,
    count(*) FILTER (WHERE ok)::bigint AS successes,
    count(*) FILTER (WHERE ok AND empty)::bigint AS empties,
    COALESCE(percentile_disc(0.95) WITHIN GROUP (ORDER BY latency_ms), 0)::bigint AS p95_latency_ms
FROM w;
