-- name: InsertCertificateMaterial :exec
-- Capture one leaf certificate's raw CT inputs into the immutable side store (spec
-- §5.3): the leaf DER and the out-of-cert SCT material, keyed by the leaf fingerprint.
-- Deduped and immutable — many Endpoints present the same certificate, so ON CONFLICT
-- DO NOTHING keeps the first capture and never rewrites a row. This writes no facet
-- value; the `certificate` observation still records only the fingerprint (ADR-0027).
INSERT INTO certificate_material (fingerprint, der, scts)
VALUES ($1, $2, $3)
ON CONFLICT (fingerprint) DO NOTHING;
