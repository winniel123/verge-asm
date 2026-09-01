-- name: InsertEdgeFanoutObservation :exec
-- Record one measured candidate edge on its Batch (ADR-0129 §6, #983): the address
-- measured, the leaf's closed outcome, the served certificate's fingerprint on
-- `presented` alone, and the handshake instant. One row per measured address.
--
-- It writes no facet value and opens no timeline — this leaf decides membership, so
-- there is no subject for the `observation` table's four-part key to name. The
-- certificate DER rides the same Batch into `certificate_material` under the same
-- fingerprint, so the SAN set the fan-out reduction counts is derived at read (#984)
-- and never copied here. An address the Scan did not measure carries no row at all:
-- an absence is never a value, and the missing row is what the veto reads as
-- *measurement pending* (#985).
INSERT INTO edge_fanout_observation (batch_id, address, outcome, fingerprint, measured_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ListEdgeFanoutMeasurements :many
-- The newest `edge-fanout` measurement per address, with the certificate the edge
-- presented beside it. This is the ONE read path from the leaf's store to the `Custody`
-- derivation's extension-reach veto (#985, ADR-0129 §4).
--
-- DISTINCT ON takes the newest row per address, and `id` breaks a tie between two rows
-- sharing a `measured_at` instant — the order the migration's index is built for. Only
-- the newest row is read: an edge measured as shared last month and dedicated today is
-- decided by today's handshake, so a veto lifts as soon as a measurement contradicts it.
--
-- The join to `certificate_material` is a LEFT JOIN and `der` is nullable, because the
-- three negative outcomes carry no fingerprint at all. A negative measured the address
-- and found no identity there, which reduces to a fan-out of zero — measured and
-- not-shared, never pending. The SAN set is derived from the DER at read (ADR-0027,
-- #983), so no SAN text is stored anywhere.
--
-- Every address the Scan measured is returned, whether or not the extension still
-- reaches it. The caller keys the derivation's input on the address, and an address no
-- longer cited by an in-zone name simply never reaches the lookup.
SELECT DISTINCT ON (o.address)
    o.address,
    o.outcome,
    m.der
FROM edge_fanout_observation o
LEFT JOIN certificate_material m ON m.fingerprint = o.fingerprint
ORDER BY o.address, o.measured_at DESC, o.id DESC;
