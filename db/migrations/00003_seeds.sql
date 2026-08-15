-- +goose Up
-- A Seed is the operator's Declared assertion of where the estate ends (v1
-- spec §3.2). Name and address scopes are genuinely different downstream — an
-- address scope enumerates every address inside it, a name scope enumerates
-- nothing — so they are distinct typed columns under one kind, not one value
-- column wearing a type flag. The native cidr type carries the address scope so
-- the database validates it and later tickets can do containment on it.
CREATE TABLE seed (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind         TEXT NOT NULL CHECK (kind IN ('name', 'address')),
    name_domain  TEXT,
    address_cidr CIDR,
    created_by   BIGINT NOT NULL REFERENCES account (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Exactly one scope column is populated, matching kind.
    CONSTRAINT seed_shape CHECK (
        (kind = 'name' AND name_domain IS NOT NULL AND address_cidr IS NULL)
        OR (kind = 'address' AND address_cidr IS NOT NULL AND name_domain IS NULL)
    )
);

-- A scope is declared once. NULLs do not collide, so one plain unique index per
-- column is enough to reject a duplicate name or address declaration.
CREATE UNIQUE INDEX seed_name_domain_key ON seed (name_domain);
CREATE UNIQUE INDEX seed_address_cidr_key ON seed (address_cidr);

-- +goose Down
DROP TABLE seed;
