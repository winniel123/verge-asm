-- +goose Up
-- The `edge-fanout` Scan — the seventh (CONTEXT.md `Scan`, ADR-0129 §6, ticket #983).
-- It is the no-SNI TLS handshake that reads the certificate a candidate edge serves to
-- a client that names nothing. The leaf shipped in #982; this migration is the Scan
-- that fires it and the store that holds what it measured.
--
-- The kind CHECK is a closed union grown one member at a time (18801, widened for `ct`
-- in 21100, for `http-identity` in 23400, for `ct-tail` in 23900). It is widened here
-- rather than upstream so the kind set travels with the dispatch that introduces it.
ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct', 'http-identity',
                    'ct-tail', 'edge-fanout'));

-- Ships ENABLED at daily cadence (86400 s) — the `dns` cadence that grants membership.
-- A slower fan-out probe would leave an edge probed as a member before the veto (#985)
-- could stop it. Its scope is the custody-extension candidates read at fan-out, so an
-- instance with no custody extension produces no job: a legible empty state (CONTEXT.md
-- `Scan`), needing no opt-in machinery. It carries no `Source` toggle and no consent
-- dial — the one handshake is a strict subset of the probing the extension already
-- authorises, run one step earlier, and it reduces total probing.
INSERT INTO scan (kind, enabled, cadence_seconds) VALUES ('edge-fanout', TRUE, 86400);

-- One row per measured address. The `edge-fanout` leaf decides MEMBERSHIP and opens no
-- timeline, so it has no facet, no subject and no discriminator — there is nothing for
-- the `observation` table's four-part key to name, and it holds no row there. This is
-- that leaf's own store.
--
--   batch_id     the Batch that measured it. The result is recorded on the Batch by
--                content, exactly as CONTEXT.md `Scan` states.
--   address      the edge address measured, in its netip form, so a lookup never turns
--                on a spelling. The port and transport are the leaf's fixed 443/tcp and
--                are not a per-row datum.
--   outcome      the leaf's closed union. An absence is never a value: a candidate the
--                Scan did not measure carries NO row, which is the state the `Custody`
--                derivation reads as measurement pending (#985).
--   fingerprint  the served leaf's `sha256:<hex>`, on `presented` alone. The certificate
--                DER itself lands in the `certificate_material` side store under this
--                same key, so the SAN set the fan-out reduction counts is derived AT
--                READ (#984) and is never copied into this row.
--   measured_at  the instant of the handshake.
CREATE TABLE edge_fanout_observation (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    batch_id    BIGINT NOT NULL REFERENCES batch (id),
    address     TEXT NOT NULL,
    outcome     TEXT NOT NULL CHECK (outcome IN ('presented', 'tls-refused', 'no-tls', 'unreachable')),
    fingerprint TEXT,
    measured_at TIMESTAMPTZ NOT NULL,
    -- A fingerprint rides `presented` and nothing else. The three negatives are each a
    -- value in their own right, never a half-read certificate, so the union is held
    -- closed by the schema rather than by the writer alone.
    CONSTRAINT edge_fanout_fingerprint_check
        CHECK ((outcome = 'presented') = (fingerprint IS NOT NULL))
);

-- The derivation reads the newest measurement per address (#984); id breaks a tie
-- between two rows sharing a measured_at instant.
CREATE INDEX edge_fanout_observation_address_time
    ON edge_fanout_observation (address, measured_at DESC, id DESC);
CREATE INDEX edge_fanout_observation_batch_idx ON edge_fanout_observation (batch_id);

-- +goose Down
DROP TABLE edge_fanout_observation;

DELETE FROM scan WHERE kind = 'edge-fanout';

ALTER TABLE scan DROP CONSTRAINT scan_kind_check;
ALTER TABLE scan ADD CONSTRAINT scan_kind_check
    CHECK (kind IN ('hot', 'cold', 'tls-acceptance', 'zone', 'dns', 'ct', 'http-identity',
                    'ct-tail'));
