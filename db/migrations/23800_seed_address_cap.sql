-- +goose Up
-- The operator-configurable address-scope cap (#888 / Settings #206, ADR-0127). It
-- bounds how many addresses one address-scope Seed (a CIDR) may cover, checked at
-- declaration per scope (ADR-0047 §5.3). ADR-0127 removes the UPPER bound on this
-- knob: nothing gates a value above the operator's own cap, so the column carries no
-- ceiling of its own, and the default stays 1024 (seed.DefaultAddressCap) so an
-- untouched install behaves identically. It lives as a column on the instance_config
-- singleton beside the other operator-global flags — migration 23000's rule that a
-- feature reads and writes columns on this one row, never mints a second store. The
-- NOT NULL DEFAULT fills the one existing row, so the empty state reads the seeded
-- default. updated_by/at record the dated act of the current cap (who last set it,
-- and when); both null while it has never been changed off the default.
ALTER TABLE instance_config
    ADD COLUMN seed_address_cap            BIGINT NOT NULL DEFAULT 1024,
    ADD COLUMN seed_address_cap_updated_by BIGINT REFERENCES account (id),
    ADD COLUMN seed_address_cap_updated_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE instance_config
    DROP COLUMN seed_address_cap,
    DROP COLUMN seed_address_cap_updated_by,
    DROP COLUMN seed_address_cap_updated_at;
