-- +goose Up
-- R4-R2 (#752): deleting a Seed returned a 500. Withdrawing a Seed is the Scope
-- chip-remove act (#21a) — a hard DELETE FROM seed — but three tables reference
-- seed(id) with the default NO ACTION, so a Seed that had ever been used could
-- not be deleted: Postgres raised a foreign_key_violation and deleteSeed
-- surfaced it as a 500. The realistic shapes that tripped it: a name scope with
-- an uploaded zone file (zone_file), a name scope CT had admitted names under
-- (admitted_name), or an address scope confirmed from a Proposal (proposal).
-- cold_scan_scope already cascades (20000_cold_scan.sql); bring the other three
-- to the same rule so a Seed withdrawal cleanly removes everything owned by /
-- derived under that Seed rather than being blocked by it.

-- zone_file: the operator's supplied ground-truth FOR a name scope. With the
-- scope withdrawn there is no scope for the supply act to restate, so it goes
-- with the Seed.
ALTER TABLE zone_file DROP CONSTRAINT zone_file_seed_id_fkey;
ALTER TABLE zone_file ADD CONSTRAINT zone_file_seed_id_fkey
    FOREIGN KEY (seed_id) REFERENCES seed (id) ON DELETE CASCADE;

-- admitted_name: how a Name entered under this name scope (a CT admission whose
-- chain terminates at the covering Seed). Withdraw the Seed and the admissions
-- that terminated at it go with it — nothing new is admitted under a scope that
-- no longer exists.
ALTER TABLE admitted_name DROP CONSTRAINT admitted_name_seed_id_fkey;
ALTER TABLE admitted_name ADD CONSTRAINT admitted_name_seed_id_fkey
    FOREIGN KEY (seed_id) REFERENCES seed (id) ON DELETE CASCADE;

-- proposal.confirmed_seed_id: the Proposal a confirmed address scope was created
-- from — provenance one hop above the Seed (ADR-0012). When the Seed is
-- withdrawn its originating Proposal is removed with it: the proposal_confirm_shape
-- CHECK forbids nulling the link while the Proposal is confirmed, so the row is
-- deleted, not detached.
ALTER TABLE proposal DROP CONSTRAINT proposal_confirmed_seed_id_fkey;
ALTER TABLE proposal ADD CONSTRAINT proposal_confirmed_seed_id_fkey
    FOREIGN KEY (confirmed_seed_id) REFERENCES seed (id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE proposal DROP CONSTRAINT proposal_confirmed_seed_id_fkey;
ALTER TABLE proposal ADD CONSTRAINT proposal_confirmed_seed_id_fkey
    FOREIGN KEY (confirmed_seed_id) REFERENCES seed (id);

ALTER TABLE admitted_name DROP CONSTRAINT admitted_name_seed_id_fkey;
ALTER TABLE admitted_name ADD CONSTRAINT admitted_name_seed_id_fkey
    FOREIGN KEY (seed_id) REFERENCES seed (id);

ALTER TABLE zone_file DROP CONSTRAINT zone_file_seed_id_fkey;
ALTER TABLE zone_file ADD CONSTRAINT zone_file_seed_id_fkey
    FOREIGN KEY (seed_id) REFERENCES seed (id);
