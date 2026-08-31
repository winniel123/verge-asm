-- +goose Up
-- The certificate-material side store (spec docs/spec/ct-source-replacement.md §5.3,
-- map #854). Verification (#878) needs the raw CT inputs a handshake carries — the SCTs
-- and the leaf certificate bytes — but ADR-0027 fences the `certificate` facet value to
-- the presented chain's fingerprints alone. So the material lands OUTSIDE the facet
-- value, in an immutable side store keyed by the leaf's fingerprint. The observation
-- still records only the fingerprint; no CT input feeds the facet value, so the fence
-- stays closed (ADR-0027 "Certificate held as an immutable value, shared by
-- fingerprint").
--
-- One row per DISTINCT certificate, deduped on the fingerprint PK: many Endpoints
-- present the same leaf, so the writer upserts ON CONFLICT DO NOTHING and the first
-- capture wins. The row is immutable — never updated, never aged.
--
--   fingerprint  the leaf DER's `sha256:<hex>` (connectoutcome.Fingerprint), the same
--                value the facet's chain[0] carries, so a chain fingerprint joins here.
--   der          the leaf certificate DER bytes. Embedded SCTs ride INSIDE this.
--   scts         the out-of-cert SCT material, captured at handshake and serialized by
--                wire.EncodeSCTCapture (TLS-extension SCTs + the stapled OCSP response).
--                NULL when the handshake carried neither.
CREATE TABLE certificate_material (
    fingerprint TEXT PRIMARY KEY,
    der         BYTEA NOT NULL,
    scts        BYTEA
);

-- +goose Down
DROP TABLE certificate_material;
