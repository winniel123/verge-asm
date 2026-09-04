-- name: InsertTranscript :exec
INSERT INTO transcript (
    queue_job_id, kind, duration_ns, captured_at, variant, outcome,
    stdout, stderr, sent_scope, truncation
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetTranscriptByJob :one
SELECT queue_job_id, kind, duration_ns, captured_at, variant, outcome,
       stdout, stderr, sent_scope, truncation, created_at
FROM transcript
WHERE queue_job_id = $1;
