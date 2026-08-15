-- +goose Up
-- An exclusion draws the estate boundary inwards (v1 spec §3.2, §6.4): an exact
-- name, a name subtree, or an address scope the operator declares is not theirs.
-- It is a Declared act with no timeline — editing one is a new exclusion — so
-- there is no status, expiry or author column, only the act's instant.
--
-- Exclusions are standalone rather than a child of a specific seed. A name
-- exclusion is a global "not mine" claim tested against the estate by label-wise
-- suffix comparison, and an address exclusion by family-matched prefix
-- comparison; neither needs a parent-seed FK to be applied, and binding one would
-- force a parent-selection rule the spec does not draw. Excluding a name that
-- still resolves — or one under no declared scope — is legal, which a FK would
-- wrongly forbid.
--
-- The exact/subtree distinction lives in kind, not in the stored name: both hold
-- the label sequence in the same form a Name's key does, and the containment test
-- a later ticket runs reads kind to decide equality vs suffix.
CREATE TABLE exclusion (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind         TEXT NOT NULL CHECK (kind IN ('name', 'subtree', 'address')),
    name         TEXT,
    address_cidr CIDR,
    created_by   BIGINT NOT NULL REFERENCES account (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- A name/subtree exclusion carries a name; an address exclusion carries a
    -- CIDR. Exactly one is populated, matching kind.
    CONSTRAINT exclusion_shape CHECK (
        (kind IN ('name', 'subtree') AND name IS NOT NULL AND address_cidr IS NULL)
        OR (kind = 'address' AND address_cidr IS NOT NULL AND name IS NULL)
    )
);

-- An exclusion is declared once. A name and a subtree of the same string are
-- different claims, so the name uniqueness keys on (kind, name); NULLs do not
-- collide, so address rows sit out of that index and the CIDR has its own.
CREATE UNIQUE INDEX exclusion_name_key ON exclusion (kind, name);
CREATE UNIQUE INDEX exclusion_address_cidr_key ON exclusion (address_cidr);

-- +goose Down
DROP TABLE exclusion;
