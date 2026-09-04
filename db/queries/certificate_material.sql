-- name: InsertCertificateMaterial :exec
INSERT INTO certificate_material (fingerprint, der, scts, issuer_spki)
VALUES ($1, $2, $3, $4)
ON CONFLICT (fingerprint) DO NOTHING;

-- name: CountCertificateMaterial :one
SELECT count(*)::bigint AS captured FROM certificate_material;

-- name: GetCertificateMaterial :one
SELECT fingerprint, der, scts, issuer_spki
FROM certificate_material
WHERE fingerprint = $1;

-- name: ListCertificateMaterialDER :many
SELECT fingerprint, der
FROM certificate_material
WHERE fingerprint = ANY(@fingerprints::text[]);
