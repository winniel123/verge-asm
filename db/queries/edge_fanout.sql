-- name: InsertEdgeFanoutObservation :exec
INSERT INTO edge_fanout_observation (batch_id, address, outcome, fingerprint, measured_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListEdgeFanoutMeasurements :many
SELECT DISTINCT ON (o.address)
    o.address,
    o.outcome,
    o.fingerprint
FROM edge_fanout_observation o
ORDER BY o.address, o.measured_at DESC, o.id DESC;

-- name: ListEdgeFanoutMeasurementsOver :many
SELECT DISTINCT ON (o.address)
    o.address,
    o.outcome,
    o.fingerprint
FROM edge_fanout_observation o
WHERE o.address = ANY(@addresses::text[])
ORDER BY o.address, o.measured_at DESC, o.id DESC;
