-- name: InsertCertificateMaterial :exec
-- Capture one leaf certificate's raw CT inputs into the immutable side store (spec
-- §5.3): the leaf DER, the out-of-cert SCT material, and the issuer SubjectPublicKeyInfo,
-- keyed by the leaf fingerprint. Deduped and immutable — many Endpoints present the same
-- certificate, so ON CONFLICT DO NOTHING keeps the first capture and never rewrites a row.
-- This writes no facet value; the `certificate` observation still records only the
-- fingerprint (ADR-0027).
INSERT INTO certificate_material (fingerprint, der, scts, issuer_spki)
VALUES ($1, $2, $3, $4)
ON CONFLICT (fingerprint) DO NOTHING;

-- name: CountCertificateMaterial :one
-- How many leaf certificates the handshake capture has stored (#881, spec §5, §6.2). The
-- More-CT-capabilities card states verification's readout: this is the pool of leaves the
-- point-check verifies against CT. Verification keeps no durable result — its logged /
-- NOT-logged findings are ephemeral events (#878) — so this captured count is the truthful
-- measure of verification's reach. One scalar row always returns.
SELECT count(*)::bigint AS captured FROM certificate_material;

-- name: GetCertificateMaterial :one
-- Read one leaf's captured CT inputs back for an on-demand verification re-check (spec §5.4,
-- #878): the leaf DER (embedded SCTs ride inside it), the out-of-cert SCT material, and the
-- issuer SubjectPublicKeyInfo the precert leaf hash needs. Keyed by the leaf fingerprint.
-- Errors with pgx.ErrNoRows when the certificate was never captured — a verification the
-- caller reports as unverifiable rather than as not-logged.
SELECT fingerprint, der, scts, issuer_spki
FROM certificate_material
WHERE fingerprint = $1;

-- name: ListCertificateMaterialDER :many
-- Read the leaf DER of each named certificate, over a fingerprint SET (#1035). This is
-- the second step of the `edge-fanout` read: ListEdgeFanoutMeasurements returns one
-- fingerprint per measured address, and this returns each DISTINCT certificate ONCE,
-- whatever number of addresses presented it.
--
-- It is not GetCertificateMaterial in a loop, and it returns the DER alone. The SCT
-- material and the issuer SubjectPublicKeyInfo are CT verification's inputs (spec §5.4);
-- the fan-out reduction reads the dNSName SANs off the leaf and nothing else, so
-- carrying them here would put bytes on the wire the caller drops.
--
-- A fingerprint with NO captured material returns NO ROW. That absence is a value, not
-- an error: a `presented` handshake whose material never landed yields no names, so its
-- edge reduces to a fan-out of zero and is reached (ADR-0129 §2). The caller must not
-- read a missing row as *measurement pending* — the missing MEASUREMENT row is what
-- means that, and it is a different absence.
SELECT fingerprint, der
FROM certificate_material
WHERE fingerprint = ANY(@fingerprints::text[]);
