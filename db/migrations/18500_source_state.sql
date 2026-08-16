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
-- upsert of the one current value, not a new row, so there is no history to
-- read. It also carries NO actor and NO instant of its own: ADR-0073 rules that
-- no operator act is written down with an actor on it, and ADR-0093 that only an
-- Annotation carries its own instant — every other Declared act, a source toggle
-- among them, is dated by the Batch whose recorded source set it moved. So the
-- row holds only the slug and the overridden state.
CREATE TABLE source_state (
    slug    TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL
);

-- +goose Down
DROP TABLE source_state;
