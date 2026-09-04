-- name: GetCTLogCursor :one
SELECT tree_size, signed_head
FROM ct_log_cursor
WHERE log_id = $1;

-- name: CTTailLastBatch :one
SELECT
    (SELECT b.created_at FROM batch b
     WHERE b.kind = 'ct-tail'
     ORDER BY b.created_at DESC, b.id DESC
     LIMIT 1) AS last_at,
    COALESCE((
        SELECT count(*)
        FROM admitted_name an
        WHERE an.batch_id = (
            SELECT b.id FROM batch b
            WHERE b.kind = 'ct-tail'
            ORDER BY b.created_at DESC, b.id DESC
            LIMIT 1
        )
    ), 0)::bigint AS names;

-- name: AdvanceCTLogCursor :exec
INSERT INTO ct_log_cursor (log_id, tree_size, signed_head, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (log_id) DO UPDATE
    SET tree_size   = EXCLUDED.tree_size,
        signed_head = EXCLUDED.signed_head,
        updated_at  = now()
    WHERE EXCLUDED.tree_size >= ct_log_cursor.tree_size;
