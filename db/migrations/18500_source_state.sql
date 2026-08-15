-- +goose Up
-- A Source's on/off state is a Declared act (v1 spec §3.1, §6.4): the operator
-- says whether a source may run. The catalogue itself — a source's identity,
-- its authority/completeness/consent, and the state it *ships* in — is authored
-- by the project and the same for every install (ADR-0003, ADR-0023: consent
-- names the door, authored in the release), so it lives in the binary rather
-- than the database. What varies per install is the operator's override of that
-- shipped default, and that is the only thing this table holds.
--
-- A row exists only where the operator has toggled a source away from — or
-- explicitly back to — its shipped default; the effective state is the override
-- where one exists and the authored default otherwise. Keying on the source's
-- stable slug (never a display name, which is a rendering) keeps the catalogue
-- free to move a label without stranding a row.
--
-- Like every Declared term the toggle carries no timeline: re-toggling is an
-- upsert of the one current value, not a new row, so there is no history to read
-- and the toggled_at is simply when the current state was last set. It is dated
-- by the act rather than left undated because a source toggle is one of the
-- Declared acts CONTEXT.md dates by what it moved (the Batch's recorded source
-- set); the instant is recorded here so a later measurement ticket can cite it.
CREATE TABLE source_state (
    slug       TEXT PRIMARY KEY,
    enabled    BOOLEAN NOT NULL,
    toggled_by BIGINT NOT NULL REFERENCES account (id),
    toggled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE source_state;
