-- +goose Up
-- The attribution columns stop pinning the account (docs/spec/audit-act.md §9).
--
-- Fifteen columns across twelve tables carry a created_by/updated_by FK to account.
-- Fourteen restrict, so an account that declared a live scope, channel or provider
-- could not be removed at all. The fifteenth already ships ON DELETE SET NULL, and
-- 24700_seed_withdrawal.sql:32-38 wrote the rule this migration generalises: the
-- attribution is worth keeping while the account exists and is not worth making a
-- member undeletable.
--
-- What changed is not the argument but its ground. The `Act` corpus captures the
-- username as a value (§5.4), so removing an account can neither orphan the record
-- of what it did nor be blocked by it. Before the corpus, dropping the FK dropped
-- the only trace.
--
-- Uniform across all fourteen (§9.4). Five are already nullable, so DROP NOT NULL
-- is a no-op there and the fourteen read the same way. No column is deleted: the
-- nine unrendered ones record acts the corpus does not cover (§9.4).
--
-- ON DELETE SET NULL alone would make each object VANISH from its listing, because
-- every attribution JOIN is an INNER JOIN. The JOIN sweep in db/queries/ lands in
-- the same change and may never be split from this one (§9.1).
--
-- The upgrade residual is named and refused (§9.5). On an upgraded install the
-- acts recorded before the corpus existed were never observed, so no backfill is
-- possible and inventing rows for them is the phantom row §7.6 bars.
ALTER TABLE seed
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT seed_created_by_fkey,
    ADD CONSTRAINT seed_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE exclusion
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT exclusion_created_by_fkey,
    ADD CONSTRAINT exclusion_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE vantage
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT vantage_created_by_fkey,
    ADD CONSTRAINT vantage_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE verge_core_frequency_edit
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT verge_core_frequency_edit_created_by_fkey,
    ADD CONSTRAINT verge_core_frequency_edit_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE cold_scan_scope
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT cold_scan_scope_created_by_fkey,
    ADD CONSTRAINT cold_scan_scope_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE zone_file
    ALTER COLUMN uploaded_by DROP NOT NULL,
    DROP CONSTRAINT zone_file_uploaded_by_fkey,
    ADD CONSTRAINT zone_file_uploaded_by_fkey FOREIGN KEY (uploaded_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE channel
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT channel_created_by_fkey,
    ADD CONSTRAINT channel_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE retention_settings
    ALTER COLUMN updated_by DROP NOT NULL,
    DROP CONSTRAINT retention_settings_updated_by_fkey,
    ADD CONSTRAINT retention_settings_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES account (id) ON DELETE SET NULL;

-- §9.2 rests every rendered by-clause on the instant beside it: a NULL author under a
-- non-NULL instant means the author was removed. retention_settings is the one site where
-- that premise did not hold, because updated_at is NOT NULL DEFAULT now() and 20600 seeds
-- the singleton, so a never-moved install carries an instant nobody wrote. The dial would
-- then name the installer "a removed account". The instant becomes nullable, and the seeded
-- one is cleared: UpdateRetentionSettings writes updated_by and updated_at together, so a
-- NULL author at this point is the never-moved row and nothing else.
ALTER TABLE retention_settings ALTER COLUMN updated_at DROP NOT NULL;
UPDATE retention_settings SET updated_at = NULL WHERE updated_by IS NULL;

ALTER TABLE proposer_lookup
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT proposer_lookup_created_by_fkey,
    ADD CONSTRAINT proposer_lookup_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE report_schedule
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT report_schedule_created_by_fkey,
    ADD CONSTRAINT report_schedule_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE sso_provider
    ALTER COLUMN created_by DROP NOT NULL,
    DROP CONSTRAINT sso_provider_created_by_fkey,
    ADD CONSTRAINT sso_provider_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE instance_config
    ALTER COLUMN api_updated_by DROP NOT NULL,
    DROP CONSTRAINT instance_config_api_updated_by_fkey,
    ADD CONSTRAINT instance_config_api_updated_by_fkey FOREIGN KEY (api_updated_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE instance_config
    ALTER COLUMN update_check_updated_by DROP NOT NULL,
    DROP CONSTRAINT instance_config_update_check_updated_by_fkey,
    ADD CONSTRAINT instance_config_update_check_updated_by_fkey FOREIGN KEY (update_check_updated_by) REFERENCES account (id) ON DELETE SET NULL;

ALTER TABLE instance_config
    ALTER COLUMN seed_address_cap_updated_by DROP NOT NULL,
    DROP CONSTRAINT instance_config_seed_address_cap_updated_by_fkey,
    ADD CONSTRAINT instance_config_seed_address_cap_updated_by_fkey FOREIGN KEY (seed_address_cap_updated_by) REFERENCES account (id) ON DELETE SET NULL;

-- +goose Down
-- The Down cannot restore NOT NULL, because rows whose author was removed now hold
-- NULL and no value exists to put back.
ALTER TABLE seed
    DROP CONSTRAINT seed_created_by_fkey,
    ADD CONSTRAINT seed_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE exclusion
    DROP CONSTRAINT exclusion_created_by_fkey,
    ADD CONSTRAINT exclusion_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE vantage
    DROP CONSTRAINT vantage_created_by_fkey,
    ADD CONSTRAINT vantage_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE verge_core_frequency_edit
    DROP CONSTRAINT verge_core_frequency_edit_created_by_fkey,
    ADD CONSTRAINT verge_core_frequency_edit_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE cold_scan_scope
    DROP CONSTRAINT cold_scan_scope_created_by_fkey,
    ADD CONSTRAINT cold_scan_scope_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE zone_file
    DROP CONSTRAINT zone_file_uploaded_by_fkey,
    ADD CONSTRAINT zone_file_uploaded_by_fkey FOREIGN KEY (uploaded_by) REFERENCES account (id);

ALTER TABLE channel
    DROP CONSTRAINT channel_created_by_fkey,
    ADD CONSTRAINT channel_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE retention_settings
    DROP CONSTRAINT retention_settings_updated_by_fkey,
    ADD CONSTRAINT retention_settings_updated_by_fkey FOREIGN KEY (updated_by) REFERENCES account (id);

UPDATE retention_settings SET updated_at = now() WHERE updated_at IS NULL;
ALTER TABLE retention_settings ALTER COLUMN updated_at SET NOT NULL;

ALTER TABLE proposer_lookup
    DROP CONSTRAINT proposer_lookup_created_by_fkey,
    ADD CONSTRAINT proposer_lookup_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE report_schedule
    DROP CONSTRAINT report_schedule_created_by_fkey,
    ADD CONSTRAINT report_schedule_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE sso_provider
    DROP CONSTRAINT sso_provider_created_by_fkey,
    ADD CONSTRAINT sso_provider_created_by_fkey FOREIGN KEY (created_by) REFERENCES account (id);

ALTER TABLE instance_config
    DROP CONSTRAINT instance_config_api_updated_by_fkey,
    ADD CONSTRAINT instance_config_api_updated_by_fkey FOREIGN KEY (api_updated_by) REFERENCES account (id);

ALTER TABLE instance_config
    DROP CONSTRAINT instance_config_update_check_updated_by_fkey,
    ADD CONSTRAINT instance_config_update_check_updated_by_fkey FOREIGN KEY (update_check_updated_by) REFERENCES account (id);

ALTER TABLE instance_config
    DROP CONSTRAINT instance_config_seed_address_cap_updated_by_fkey,
    ADD CONSTRAINT instance_config_seed_address_cap_updated_by_fkey FOREIGN KEY (seed_address_cap_updated_by) REFERENCES account (id);
