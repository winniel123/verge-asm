-- +goose Up
-- A Break is derived on read, never stored (ADR-0007, ADR-0008), so the Coverage panel reads
-- it as a lag() over span partitioned on the timeline key and ordered by opened_at. Neither
-- index from 19000 can serve that window: span_open_timeline_idx is partial on
-- `closed_at IS NULL`, so it holds one row per timeline rather than the whole history, and
-- span_subject_idx leads with subject_kind, which the partition does not name — a leading
-- column the window does not group on cannot supply its order. Without this index every
-- Coverage load sorts the whole span corpus (#1768).
CREATE INDEX span_timeline_opened_idx
    ON span (subject_key, facet, discriminator, vantage_id, source, opened_at);

-- +goose Down
DROP INDEX span_timeline_opened_idx;
