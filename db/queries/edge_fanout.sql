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
