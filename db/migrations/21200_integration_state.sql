-- +goose Up
-- An Integration's install state is a Declared act (v1 spec §6.1): the operator
-- says whether a third-party integration is installed against this deployment. The
-- integration catalogue itself — a tile's identity, category, description, and the
-- consent grants it would receive — is authored by the project and the same for
-- every install, so it lives in the binary rather than the database (mirroring
-- source_state / ADR-0003, ADR-0023: consent names the door, authored in the
-- release). What varies per install is the operator's choice to install one, and
-- the state that install is in, and that is the only thing this table holds.
--
-- A row exists only where the operator has installed an integration; an
-- integration with no row is available (not installed). The effective state is the
-- stored state where a row exists and available otherwise. Keying on the
-- integration's stable slug (never a display name, which is a rendering) keeps the
-- catalogue free to move a label without stranding a row.
--
-- Like every Declared term the install carries no timeline, no actor, and no
-- instant of its own (ADR-0073, ADR-0093): re-installing is an upsert of the one
-- current state, disconnecting deletes the row, and neither writes a history to
-- read. An integration is NOT a channel and NOT a source: it is a third-party
-- install tile, distinct from a delivery channel (which carries messages) and from
-- a discovery source (which observes) — so it keeps its own table, never folded
-- into channels or source_state.
CREATE TABLE integration_state (
    slug  TEXT PRIMARY KEY,
    state TEXT NOT NULL
);

-- +goose Down
DROP TABLE integration_state;
