-- +goose Up
-- The NAME limb of the Seed tombstone (ADR-0135 §2, ticket #1045). ADR-0134 §7
-- named it and left it out, so 24700 shipped `address_cidr CIDR NOT NULL`.
--
-- The tombstone takes `seed`'s own shape (00003_seeds.sql): one table, a `kind`
-- discriminator, one scope column per limb, and a CHECK that exactly one is
-- populated. A tombstone is the counterpart act to a declaration, so it is
-- recorded the way the declaration is. A second table would duplicate the
-- consumed_at machinery and give each fold its own spend query for no gain.
--
-- DEFAULT 'address' backfills the rows 24700 wrote, every one of which is an
-- address withdrawal. The default is then dropped so a new insert must state its
-- kind rather than inherit one.
ALTER TABLE seed_withdrawal ADD COLUMN kind TEXT NOT NULL DEFAULT 'address'
    CHECK (kind IN ('name', 'address'));
ALTER TABLE seed_withdrawal ALTER COLUMN kind DROP DEFAULT;

ALTER TABLE seed_withdrawal ADD COLUMN name_domain TEXT;
ALTER TABLE seed_withdrawal ALTER COLUMN address_cidr DROP NOT NULL;

ALTER TABLE seed_withdrawal ADD CONSTRAINT seed_withdrawal_shape CHECK (
    (kind = 'name' AND name_domain IS NOT NULL AND address_cidr IS NULL)
    OR (kind = 'address' AND address_cidr IS NOT NULL AND name_domain IS NULL)
);

-- The domain alone. The `seed` delete cascades away the `admitted_name` and
-- `zone_file` rows that admitted the Names, but it touches no `span`, and `span`
-- is where the fold reads its candidates. The evidence that admitted a Name is
-- not the record of its timelines (ADR-0135 §4).
--
-- Each fold now reads only its own limb, so `kind` leads the pending index.
DROP INDEX seed_withdrawal_pending_idx;
CREATE INDEX seed_withdrawal_pending_idx ON seed_withdrawal (kind, id) WHERE consumed_at IS NULL;

-- +goose Down
DROP INDEX seed_withdrawal_pending_idx;
CREATE INDEX seed_withdrawal_pending_idx ON seed_withdrawal (id) WHERE consumed_at IS NULL;

ALTER TABLE seed_withdrawal DROP CONSTRAINT seed_withdrawal_shape;
DELETE FROM seed_withdrawal WHERE kind = 'name';
ALTER TABLE seed_withdrawal ALTER COLUMN address_cidr SET NOT NULL;
ALTER TABLE seed_withdrawal DROP COLUMN name_domain;
ALTER TABLE seed_withdrawal DROP COLUMN kind;
