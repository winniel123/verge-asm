-- +goose Up
-- Seed the Cert Spotter reservation row in the per-source CT throttle
-- (spec docs/spec/ct-source-replacement.md §2.5, map #854, ticket #876). The
-- per-source ct_throttle table arrived in migration 23600 with only crt.sh's row.
-- ReserveCTSlot updates the row for its source slug and returns no slot when the
-- row is absent, so the operator-keyed Cert Spotter primary needs its row present
-- before its first reserve. This seeds it. No citation migration: admitted_name
-- rows are untouched (spec §2.7); a Cert Spotter admission is a new row citing its
-- own Batch. ON CONFLICT DO NOTHING keeps re-running the migration safe.
INSERT INTO ct_throttle (source, next_free_at)
VALUES ('certspotter', now())
ON CONFLICT (source) DO NOTHING;

-- +goose Down
DELETE FROM ct_throttle WHERE source = 'certspotter';
