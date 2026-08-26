-- +goose Up
-- The instance-global singleton (v3.18.0 consume: #390 read-only API token surfaces
-- and #391 Backup & updates). One operator-wide row of flags, who/when records and
-- cached facts that both feature clusters read and write — modelled exactly on
-- retention_settings (migration 20600): a boolean primary key can only ever hold one
-- true row, so the configuration is global and there is exactly one to read. It is
-- minted ONCE here so no two feature children collide on a second migration (the known
-- goose-number gotcha); features read and write columns, they never add a store.
CREATE TABLE instance_config (
    -- A single-row table: the boolean primary key can only ever hold one true row, so
    -- the configuration is global and there is exactly one to read.
    id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),

    -- #390 API surfaces. api_enabled is the single admin switch that makes /api/v1
    -- answer GET for valid personal tokens — off by default, so a minted token stays
    -- inert until an admin flips it. updated_by/at record the dated act of the CURRENT
    -- state (who last flipped it, and when); both null while it has never been enabled.
    api_enabled    BOOLEAN NOT NULL DEFAULT false,
    api_updated_by BIGINT REFERENCES account (id),
    api_updated_at TIMESTAMPTZ,

    -- #391 update checks. update_check_enabled opts the worker into a best-effort daily
    -- release-feed check — off by default and air-gap-safe, so no network call is ever
    -- made while disabled. updated_by/at record who last flipped it, and when.
    update_check_enabled    BOOLEAN NOT NULL DEFAULT false,
    update_check_updated_by BIGINT REFERENCES account (id),
    update_check_updated_at TIMESTAMPTZ,

    -- #391 release cache — the last result of the worker's release check, surfaced on
    -- the Instance tab's Version & updates card. All nullable: nothing has been checked
    -- yet. release_state is current | newer | disabled; the latest version and notes
    -- accompany a "newer".
    release_state          TEXT,
    release_latest_version TEXT,
    release_latest_notes   TEXT,
    release_checked_at     TIMESTAMPTZ,

    -- #391 last UI-taken backup record — surfaced on the Backup card. Both null until a
    -- backup has been streamed from the UI.
    last_backup_at   TIMESTAMPTZ,
    last_backup_size BIGINT
);

INSERT INTO instance_config (id) VALUES (true);

-- +goose Down
DROP TABLE instance_config;
