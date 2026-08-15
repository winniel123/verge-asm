-- +goose Up
-- A custody extension is a property of a NAME scope alone (v1 spec §3.2, §6.4;
-- ADR-0013): the operator's declaration that the addresses its names resolve to
-- are inside the boundary, and therefore under their Custody. It is off by
-- default, declared once on a scope the operator already authors, and
-- withdrawable — a Declared boolean with no timeline, so declaring or withdrawing
-- it flips the flag rather than writing a dated row.
--
-- It is barred from an address scope by construction: an address scope is its own
-- complete enumeration and needs no extension, and the Vantage-class gate must
-- never read one. The CHECK holds the column false wherever kind is not 'name',
-- so a stray write cannot mint a custody extension on an address scope.
ALTER TABLE seed
    ADD COLUMN custody_extension BOOLEAN NOT NULL DEFAULT false,
    ADD CONSTRAINT seed_custody_extension_name_only
        CHECK (kind = 'name' OR custody_extension = false);

-- +goose Down
ALTER TABLE seed
    DROP CONSTRAINT seed_custody_extension_name_only,
    DROP COLUMN custody_extension;
