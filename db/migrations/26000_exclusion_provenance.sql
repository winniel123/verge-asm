-- +goose Up
-- An exclusion row records no act that made it (#1799).
--
-- A decline writes an address exclusion, and undoing that decline lifts it.
-- exclusion_address_cidr_key (00004_exclusions.sql:38) is unique on address_cidr,
-- so one CIDR holds one row whoever declared it. The decline loop tolerates the
-- unique violation, so a decline over a CIDR the operator already declared writes
-- nothing and adopts the operator's row. The undo then deleted that row. Nothing
-- told the operator, and no screen showed the declaration had ever been there.
--
-- created_by cannot answer this. It is the account on both paths and is often the
-- same operator. created_at orders the two rows and proves nothing about which act
-- made either. proposal_id is the fact neither carries: a decline's exclusion names
-- the proposal it answered, and a hand-declared one is NULL.
--
-- The delete predicate reads IS NOT NULL rather than equality with the undone
-- proposal. Two proposals may name one CIDR, the second decline's insert writes
-- nothing, and the row keeps the first proposal's id. Equality would strand that
-- row once the first decline was undone, which the sibling guard added by #1777
-- already handles correctly.
--
-- ON DELETE SET NULL, so a removed proposal leaves the exclusion standing and the
-- row falls back to the reading that deletes nothing.
ALTER TABLE exclusion
    ADD COLUMN proposal_id BIGINT REFERENCES proposal (id) ON DELETE SET NULL;

-- Only a decline of an address proposal writes an exclusion, so a name or subtree
-- row can carry no proposal.
ALTER TABLE exclusion
    ADD CONSTRAINT exclusion_provenance CHECK (
        proposal_id IS NULL OR kind = 'address'
    );

-- NO BACKFILL, and the residue is named rather than guessed. No fact in an existing
-- row separates a decline's exclusion from a hand-declared one — that absence is the
-- whole ticket — so every row written before this migration reads as hand-declared.
-- On an upgraded install a legacy decline's exclusion therefore survives its own
-- undo, and the operator lifts it on the exclusions screen, which the undo's flash
-- says. Matching on address_cidr against declined proposals would restore that undo
-- and would reproduce the fault exactly, by claiming the operator's own declaration
-- of a CIDR some proposal also named.

-- +goose Down
ALTER TABLE exclusion DROP CONSTRAINT exclusion_provenance;
ALTER TABLE exclusion DROP COLUMN proposal_id;
