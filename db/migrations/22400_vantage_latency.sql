-- +goose Up
-- Per-vantage connect latency (P0.5; SPEC-CHANGE.md collision #7). The Dashboard
-- Vantages card shows a round-trip reading beside each provisioned prober
-- (design-system/examples/console/Dashboard.jsx renders it as a mono "34ms"). The
-- datum is measured at the prober SSH connect — the same connect that pins the
-- host key trust-on-first-use — and stored here on the vantage.
--
-- It is NULLABLE on purpose: it holds no value until that first connect lands a
-- measurement, and until then the Dashboard renders the spec's pending em dash
-- rather than a fabricated number. Only a provisioned prober is ever measured, so
-- the resolver-only `local` vantage keeps this NULL like its other prober columns.
-- Stored in whole milliseconds to match exactly what the console renders.
ALTER TABLE vantage ADD COLUMN latency_ms INTEGER
    CHECK (latency_ms IS NULL OR latency_ms >= 0);

-- +goose Down
ALTER TABLE vantage DROP COLUMN latency_ms;
