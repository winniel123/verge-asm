-- +goose Up
-- Add the issuer's SubjectPublicKeyInfo to the certificate-material side store (spec
-- docs/spec/ct-source-replacement.md §5.3, map #854). Verification (#878) of an EMBEDDED SCT
-- hashes the PRECERTIFICATE, and the precert leaf hash carries
-- issuer_key_hash = SHA-256(issuer SubjectPublicKeyInfo) (RFC 6962 §3.2). The leaf DER alone
-- does not carry the issuer's key, so the handshake captures the issuer (chain position 1)
-- beside the leaf here.
--
-- NULL when the handshake presented no issuer (a lone self-signed leaf), when the capture
-- predates this column, or on a scripted golden row. Such a row still verifies a TLS-extension
-- or OCSP SCT — an x509 entry needs no issuer — only not an embedded one, which is reported
-- unverifiable rather than not-logged. The row stays immutable and deduped on the fingerprint
-- PK, exactly as the leaf DER and scts are.
ALTER TABLE certificate_material ADD COLUMN issuer_spki BYTEA;

-- +goose Down
ALTER TABLE certificate_material DROP COLUMN issuer_spki;
