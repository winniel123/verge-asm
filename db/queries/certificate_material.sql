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

-- name: GetCertificateMaterial :one
-- Read one leaf's captured CT inputs back for an on-demand verification re-check (spec §5.4,
-- #878): the leaf DER (embedded SCTs ride inside it), the out-of-cert SCT material, and the
-- issuer SubjectPublicKeyInfo the precert leaf hash needs. Keyed by the leaf fingerprint.
-- Errors with pgx.ErrNoRows when the certificate was never captured — a verification the
-- caller reports as unverifiable rather than as not-logged.
SELECT fingerprint, der, scts, issuer_spki
FROM certificate_material
WHERE fingerprint = $1;
