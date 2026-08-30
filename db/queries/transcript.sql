-- name: InsertTranscript :exec
-- Persist one job's verbatim transcript (raw-job-output spec §1.4), mirroring
-- InsertObservation. The producer calls it once per captured job inside the
-- worker's terminal transaction (§2.4), so a job cancelled mid-flight rolls its
-- transcript back with the rest of its work. A job with no capture inserts no
-- row — the absence is legible, distinct from a captured-but-empty stream.
INSERT INTO transcript (
    queue_job_id, kind, duration_ns, captured_at, variant, outcome,
    stdout, stderr, sent_scope, truncation
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetTranscriptByJob :one
-- The single transcript for a queue_job id — the §6 read handler's source for
-- `?job={id}`. One row per attempt (spec §1.1), so this addresses a transcript
-- directly. Returns no row when the job produced no capture (a legible absence,
-- which the handler renders distinctly from a captured-but-empty stream).
SELECT queue_job_id, kind, duration_ns, captured_at, variant, outcome,
       stdout, stderr, sent_scope, truncation, created_at
FROM transcript
WHERE queue_job_id = $1;
