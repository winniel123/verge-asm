-- +goose Up
-- A per-vantage reading names the vantage it came from. Until now only the
-- readers agreed on that, and they disagreed with each other (ADR-1985).
--
-- span.vantage_id and observation.vantage_id are both nullable, so a
-- `service` / `reachability` row could be stored with no vantage at all.
-- ListServiceReachabilitySpansByClass inner-joins vantage and dropped such a
-- row; the asset census (ListAllOpenSpans) filters on closed_at alone and kept
-- it. The row survived the census, its legs did not survive the class read, and
-- the asset page rendered `never looked` on BOTH legs of a port the estate had
-- measured (#1985, #1962). That is a false aperture claim, which ADR-0095 rules
-- is worse than a missing reading.
--
-- The invariant lives in the schema rather than in a render or a census filter.
-- One expression, enforced on write, and no read has to agree with any other
-- read to keep it true — the move the partial unique index in 19000 already
-- makes for "the open span is the current state" (ADR-1985 §3).
--
-- The predicate is a DENY-LIST and it fails closed. A row needs a vantage
-- UNLESS it is one of the two exceptions below, so a facet added later is
-- constrained on the day it is added and has to argue its way onto an exception
-- line. An allow-list would admit a new per-vantage facet in silence, which is
-- the failure mode that produced #1985 (ADR-1985 §4).
--
-- Neither a facet-only nor a source-only predicate can state the rule:
-- `certificate` is mixed inside one source, and `dns-record` is mixed across
-- two. Each exception therefore carries its citation at the exception.
--
-- This is NOT `NOT VALID`. No production writer can emit a violating row —
-- BuildColdJobs and BuildHotJobs both return early on an empty vantage list and
-- both stamp the loop's vantage id, and vantage_id carries a plain REFERENCES
-- with no ON DELETE SET NULL. The violating rows this change expects are the
-- inventory fixture seeder's, and a plain constraint fails their migration
-- loudly rather than staying silent about them (ADR-1985 §6).
--
-- RECOVERY IS A DROP, NOT A RE-SEED. cmd/web/main.go runs migrateUp BEFORE it
-- reads -seed-fixtures, so the rewritten seeder cannot repair a database this
-- migration refuses: the binary exits at migrate and never reaches it. The
-- observation rows are further out of the seeder's reach, because it writes and
-- clears span alone. A developer holding either drops the database volume and
-- seeds again. No migration deletes a row here: that would be the first
-- exception to ADR-0041 and needs its own argument.
ALTER TABLE span
    ADD CONSTRAINT span_per_vantage_facet_needs_vantage CHECK (
        vantage_id IS NOT NULL
        -- The zone reader restates a stored file from no network position, so
        -- its dns-record rows carry no vantage (internal/scan/zone.go, ADR-0027).
        OR source = 'zone'
        -- A default certificate is not per-vantage, so the edge fan-out nulls
        -- it (ADR-0129, #954).
        OR facet = 'certificate'
    );

-- observation carries the identical hole. ListNameResolutionsByClass inner-joins
-- vantage on an OBSERVATION, so the same two-read disagreement is reachable
-- there, and the span fold copies the observation's vantage forward unchanged.
ALTER TABLE observation
    ADD CONSTRAINT observation_per_vantage_facet_needs_vantage CHECK (
        vantage_id IS NOT NULL
        OR source = 'zone'
        OR facet = 'certificate'
    );

-- +goose Down
ALTER TABLE observation DROP CONSTRAINT observation_per_vantage_facet_needs_vantage;
ALTER TABLE span DROP CONSTRAINT span_per_vantage_facet_needs_vantage;
