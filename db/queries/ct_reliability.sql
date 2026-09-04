-- name: InsertCTReliabilitySample :exec
INSERT INTO ct_reliability_sample (source, ok, latency_ms, empty)
VALUES ($1, $2, $3, $4);

-- name: TrimCTReliabilitySamples :exec
DELETE FROM ct_reliability_sample AS s
WHERE s.source = sqlc.arg(source)
  AND s.id NOT IN (
      SELECT keep.id FROM ct_reliability_sample AS keep
      WHERE keep.source = sqlc.arg(source)
      ORDER BY keep.observed_at DESC, keep.id DESC
      LIMIT sqlc.arg(keep_count)
  );

-- name: CTReliabilityWindow :one
WITH w AS (
    SELECT ok, latency_ms, empty, observed_at
    FROM ct_reliability_sample
    WHERE source = sqlc.arg(source)
    ORDER BY observed_at DESC, id DESC
    LIMIT sqlc.arg(sample_size)
)
SELECT
    count(*)::bigint AS total,
    count(*) FILTER (WHERE ok)::bigint AS successes,
    count(*) FILTER (WHERE ok AND empty)::bigint AS empties,
    COALESCE(percentile_disc(0.95) WITHIN GROUP (ORDER BY latency_ms), 0)::bigint AS p95_latency_ms,
    MAX(observed_at)::timestamptz AS last_at
FROM w;
