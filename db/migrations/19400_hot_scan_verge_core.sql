-- +goose Up
-- The hot Scan (v1 spec §3.4/§3.5): daily, the ONLY port tier that ships
-- enabled, its scope the `verge-core` port set, dispatched as the
-- `connect-outcome` leaf and gated totally by `Custody`. Daily is the tightest
-- shipped cadence; the cold full-range tier lands disabled in a later ticket.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('hot', TRUE, 86400);

-- The reachability facet joins the closed set an observation may carry. It is
-- what the connect-outcome leaf decides for a `Service`; adding a facet is
-- strictly additive (CONTEXT.md `Facet`). The subject_kind CHECK already admits
-- 'service', so only the facet set widens here.
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record', 'reachability'));

-- verge-core's body is compiled in (ADR-0144), and ONLY the frequency half is
-- operator-editable — the sensitive half is authored by the release, because
-- moving one would move a version and `Break` the estate (§3.5). This table holds
-- the operator's edits to the frequency half as deltas over the shipped default:
-- one row per port, `add` to include a frequency port the default omits or
-- `remove` to drop one. A removed pair that is also sensitive stays probed — the
-- union keeps it — so an operator can never move the sensitive half by editing
-- the frequency one. UDP is not editable: the frequency half is TCP-only.
CREATE TABLE verge_core_frequency_edit (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    port       INTEGER NOT NULL CHECK (port BETWEEN 1 AND 65535),
    action     TEXT NOT NULL CHECK (action IN ('add', 'remove')),
    created_by BIGINT NOT NULL REFERENCES account (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One edit per port: a later edit replaces the earlier one (upsert on the port),
-- and resetting a port to its shipped default is deleting its row.
CREATE UNIQUE INDEX verge_core_frequency_edit_port_key ON verge_core_frequency_edit (port);

-- +goose Down
DROP TABLE verge_core_frequency_edit;
ALTER TABLE observation DROP CONSTRAINT observation_facet_check;
ALTER TABLE observation ADD CONSTRAINT observation_facet_check
    CHECK (facet IN ('resolution', 'dns-record'));
DELETE FROM scan WHERE kind = 'hot';
