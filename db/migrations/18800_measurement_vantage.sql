-- +goose Up
-- The unified `vantage` table is created in 18700. This migration ships the one
-- resolver-only Vantage a name-scope install needs so the dns Scan resolves at
-- the dns cadence from the first run. It has no prober — every prober column is
-- left NULL (ADR-0103) — and class ships `unverified`, the honest default until
-- a prober observes the instance's presented address (v1 spec §4.2).
--
-- The resolver is operator-editable, but the shipped value must work on the
-- deployment the docs describe (ADR-0036: a shipped default is the configuration
-- that takes effect). That deployment is `docker compose`, where the worker's
-- recursive resolver is Docker's embedded DNS — the address the INSERT below
-- ships. Off compose (bare-metal / host-network, where that address is not routed)
-- the operator points it at their own recursive resolver; docs/guides/using.md
-- "Run the first batch" and docs/guides/running.md Configuration cover that.
--
-- This loopback value is DELIBERATE and works at runtime: the measurement dial
-- path exempts the operator-declared recursive resolver from the SSRF/rebinding
-- egress custody guard, which gates only discovered walk authorities (ADR-0121,
-- #612). Do not "fix" this back to a public resolver — the guard once refused it
-- and dead-lettered every default install (a regression of #239).
INSERT INTO vantage (name, class, resolver) VALUES ('local', 'unverified', '127.0.0.11:53');

-- +goose Down
DELETE FROM vantage WHERE name = 'local';
