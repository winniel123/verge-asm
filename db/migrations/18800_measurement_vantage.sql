-- +goose Up
-- The unified `vantage` table is created in 18700. This migration ships the one
-- resolver-only Vantage a name-scope install needs so the dns Scan resolves at
-- the dns cadence from the first run. It has no prober — every prober column is
-- left NULL (ADR-0103) — and class ships `unverified`, the honest default until
-- a prober observes the instance's presented address (v1 spec §4.2). Its
-- resolver is operator-editable; the shipped value is a placeholder the operator
-- replaces with their own recursive resolver.
INSERT INTO vantage (name, class, resolver) VALUES ('local', 'unverified', '127.0.0.1:53');

-- +goose Down
DELETE FROM vantage WHERE name = 'local';
