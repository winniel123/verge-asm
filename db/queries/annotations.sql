-- Reads and writes behind `Annotation` management on the Signals screen (#204).
-- An Annotation is an operator dial keyed on one `(subject, signal-name)` pair,
-- carrying the operator's reason and the instant declared — no status, no expiry
-- and no author (CONTEXT.md `Annotation`, ADR-0073). Declaring and withdrawing
-- are plain state changes: neither is a `Message`, and neither mints a cause.

-- name: CreateAnnotation :one
-- Declare an acceptance on one pair. The unique index on (subject_key,
-- signal_name) rejects a re-declaration of the same pair — an Annotation cannot
-- be edited, so changing the reason is a withdraw-then-declare, not an update.
INSERT INTO annotation (subject_key, signal_name, reason)
VALUES ($1, $2, $3)
RETURNING id, subject_key, signal_name, reason, declared_at;

-- name: ListAnnotations :many
-- Every declared acceptance, ordered by signal then subject — a deterministic
-- list with no sort by attention, age or count (an operator dial carries no such
-- axis). The Signals layer folds these against the live census to decide the
-- fully-annotated prose case and to mark a row whose key names no current member.
SELECT id, subject_key, signal_name, reason, declared_at
FROM annotation
ORDER BY signal_name, subject_key;

-- name: DeleteAnnotation :exec
-- Withdraw an acceptance. Withdrawing is a plain state change that produces no
-- `Message` — its carrier is the message it releases, the pair's own next firing.
-- Deleting a row that is already gone is not an error: the operator's intent, that
-- the acceptance no longer stand, is satisfied either way.
DELETE FROM annotation WHERE id = $1;
