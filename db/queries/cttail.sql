-- name: GetCTLogCursor :one
-- The tail's forward cursor for one CT log (spec §4.2): the last tree size read and
-- the last signed head seen. A log with no row yet has never been polled — the caller
-- treats pgx.ErrNoRows as "start at position 0" and reads the whole current delta from
-- the log's origin forward, never backfilling below it afterwards.
SELECT tree_size, signed_head
FROM ct_log_cursor
WHERE log_id = $1;

-- name: AdvanceCTLogCursor :exec
-- Move one log's cursor forward to the tree size just read, recording the STH that
-- signed it (spec §4.2). Forward-only by construction: the ON CONFLICT update advances
-- the row only when the new tree size is at or beyond the stored one, so a stale or
-- out-of-order poll can never rewind the cursor and re-admit history (the §4 invariant).
-- The first poll of a log inserts its row.
INSERT INTO ct_log_cursor (log_id, tree_size, signed_head, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (log_id) DO UPDATE
    SET tree_size   = EXCLUDED.tree_size,
        signed_head = EXCLUDED.signed_head,
        updated_at  = now()
    WHERE EXCLUDED.tree_size >= ct_log_cursor.tree_size;
