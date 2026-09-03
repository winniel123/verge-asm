// This file builds the pure half of CT verification (spec docs/spec/ct-source-replacement.md
// §5, map #854): the point-check that confirms ONE specific certificate is logged in CT.
// It complements the drift tail (internal/scan/cttail.go): the tail catches certificates in
// CT the operator did not expect; verification catches certificates the operator observed
// that are NOT in CT — an internal CA, or evasion (§5).
//
// The design invariant is that verification only POINT-CHECKS and never enumerates. It
// always starts from an SCT or the certificate bytes, never a name (§5.1, RFC 6962 §4 has no
// query-by-name). Given a certificate this file computes the RFC 6962 MerkleTreeLeaf hash and
// hands the impure half (internal/queue/ctverify.go) what it needs to ask the correct log or
// shard: for an RFC 6962 log the leaf hash for a get-proof-by-hash request and the audit-path
// recompute to the signed tree head's root; for a static-ct-api (tiled) log the leaf hash and
// the SCT's leaf index to confirm against the level-0 hash tile. This file never touches the
// network and never verifies a log SIGNATURE — the signed-head signature stays deferred with
// the tail's (§4.4) — so it recomputes inclusion to the head's stated root, no key trust.
package scan

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"
	"strings"

	"golang.org/x/crypto/cryptobyte"
	cryptobyte_asn1 "golang.org/x/crypto/cryptobyte/asn1"
	"golang.org/x/crypto/ocsp"
)

// oidSCTList is the X.509v3 / OCSP extension carrying the embedded SCT list (RFC 6962 §3.3,
// OID 1.3.6.1.4.1.11129.2.4.2). An embedded SCT is signed over the PRECERTIFICATE, so its
// leaf hash uses the precert reconstruction (PrecertTBS + IssuerKeyHash), not the final cert.
var oidSCTList = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}

// SCT is the subset of an RFC 6962 SignedCertificateTimestamp verification reads (§3.2): the
// log the SCT names, the timestamp that binds the MerkleTreeLeaf, and the SCT's own extensions
// — empty for a classic RFC 6962 log, but carrying the leaf_index for a static-ct-api log
// (SCTLeafIndex). The digitally-signed signature is read past and discarded: verification
// recomputes inclusion to the head's root and never checks the log's signature (§4.4).
type SCT struct {
	Version uint8
	// LogID is the 32-byte SHA-256 of the log's public key — the same value log_list.json
	// carries base64-encoded, so FindLogByLogID matches on the base64 of this.
	LogID     [32]byte
	Timestamp uint64
	// Extensions is the CtExtensions content (already unwrapped from its opaque<0..2^16-1>
	// length). It rides into the TimestampedEntry.extensions when the leaf hash is built, so
	// the same bytes that named the leaf_index also bind the hash (§5.1).
	Extensions []byte
}

// ParseSCT decodes one serialized SignedCertificateTimestamp (RFC 6962 §3.2). It reads the
// version, the 32-byte log id, the 8-byte timestamp and the extensions, then reads past the
// digitally-signed signature — a hash algorithm byte, a signature algorithm byte and an
// opaque<0..2^16-1> signature — and requires the input to end there. A truncated or
// trailing-garbage SCT is an error, never a silent partial read.
func ParseSCT(b []byte) (SCT, error) {
	s := cryptobyte.String(b)
	var sct SCT
	if !s.ReadUint8(&sct.Version) {
		return SCT{}, fmt.Errorf("scan: sct version")
	}
	if !s.CopyBytes(sct.LogID[:]) {
		return SCT{}, fmt.Errorf("scan: sct log id")
	}
	var ts []byte
	if !s.ReadBytes(&ts, 8) {
		return SCT{}, fmt.Errorf("scan: sct timestamp")
	}
	sct.Timestamp = binary.BigEndian.Uint64(ts)
	var ext cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&ext) {
		return SCT{}, fmt.Errorf("scan: sct extensions")
	}
	sct.Extensions = append([]byte(nil), ext...)
	var alg []byte
	var sig cryptobyte.String
	if !s.ReadBytes(&alg, 2) || !s.ReadUint16LengthPrefixed(&sig) {
		return SCT{}, fmt.Errorf("scan: sct signature")
	}
	if !s.Empty() {
		return SCT{}, fmt.Errorf("scan: sct has %d trailing bytes", len(s))
	}
	if sct.Version != ctSCTVersionV1 {
		return SCT{}, fmt.Errorf("scan: sct version %d unsupported", sct.Version)
	}
	return sct, nil
}

// ctSCTVersionV1 is SignedCertificateTimestamp.sct_version v1 (RFC 6962 §3.2).
const ctSCTVersionV1 = 0

// SCTLeafIndex reads a static-ct-api leaf_index from an SCT's extensions (C2SP static-ct-api).
// The extensions are a sequence of Extension { uint8 type; opaque<0..2^16-1> data }; the
// leaf_index extension is type 0 with a 5-byte big-endian (uint40) index. It reports false
// when no leaf_index extension is present — a classic RFC 6962 SCT carries none, and a tiled
// log's SCT always does, so its presence also discriminates the shard's client. A malformed
// extensions block reports false rather than a wrong index.
func SCTLeafIndex(extensions []byte) (int64, bool) {
	s := cryptobyte.String(extensions)
	for !s.Empty() {
		var extType uint8
		var data cryptobyte.String
		if !s.ReadUint8(&extType) || !s.ReadUint16LengthPrefixed(&data) {
			return 0, false
		}
		if extType == ctExtLeafIndex {
			if len(data) != 5 {
				return 0, false
			}
			var idx int64
			for _, bb := range data {
				idx = idx<<8 | int64(bb)
			}
			return idx, true
		}
	}
	return 0, false
}

const ctExtLeafIndex = 0

// EmbeddedSCTs extracts the SCTs embedded in a leaf certificate's X.509v3 SCT-list extension
// (RFC 6962 §3.3). Each is signed over the PRECERTIFICATE, so its leaf hash uses the precert
// reconstruction. A certificate with no SCT-list extension yields no SCTs and no error — many
// certificates carry their SCTs only out of band (the TLS extension or an OCSP staple). The
// extension value is a DER OCTET STRING wrapping a TLS SerializedSCT list, unwrapped here.
func EmbeddedSCTs(leafDER []byte) ([][]byte, error) {
	cert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, fmt.Errorf("scan: parse leaf for embedded scts: %w", err)
	}
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(oidSCTList) {
			continue
		}
		// ext.Value is the DER-encoded extnValue: an OCTET STRING whose content is the TLS
		// SignedCertificateTimestampList (RFC 6962 §3.3). Unwrap the OCTET STRING first.
		var inner cryptobyte.String
		outer := cryptobyte.String(ext.Value)
		if !outer.ReadASN1(&inner, cryptobyte_asn1.OCTET_STRING) {
			return nil, fmt.Errorf("scan: embedded sct octet string")
		}
		return parseSCTList(inner)
	}
	return nil, nil
}

// OCSPSCTs extracts the SCTs carried in a stapled OCSP response's SingleResponse extension
// (RFC 6962 §3.3). These are signed over the FINAL certificate (an x509 entry), like the
// TLS-extension SCTs. It is best-effort: an OCSP response that does not parse, or carries no
// SCT extension, yields no SCTs and no error, so a malformed staple never fails a verification
// — it just narrows the SCTs available. The signature is not checked (nil issuer).
func OCSPSCTs(ocspResponse []byte) ([][]byte, error) {
	if len(ocspResponse) == 0 {
		return nil, nil
	}
	resp, err := ocsp.ParseResponse(ocspResponse, nil)
	if err != nil {
		return nil, nil // best-effort: an unparseable staple contributes no SCTs
	}
	for _, ext := range resp.Extensions {
		if !ext.Id.Equal(oidSCTList) {
			continue
		}
		var inner cryptobyte.String
		outer := cryptobyte.String(ext.Value)
		if !outer.ReadASN1(&inner, cryptobyte_asn1.OCTET_STRING) {
			return nil, nil
		}
		list, perr := parseSCTList(inner)
		if perr != nil {
			return nil, nil
		}
		return list, nil
	}
	return nil, nil
}

// parseSCTList decodes a TLS SignedCertificateTimestampList (RFC 6962 §3.3): an
// opaque<1..2^16-1> outer list of opaque<1..2^16-1> serialized SCTs. Each element is one
// serialized SCT for ParseSCT. It is shared by the embedded and OCSP extraction paths.
func parseSCTList(b cryptobyte.String) ([][]byte, error) {
	var list cryptobyte.String
	if !b.ReadUint16LengthPrefixed(&list) {
		return nil, fmt.Errorf("scan: sct list length")
	}
	var out [][]byte
	for !list.Empty() {
		var one cryptobyte.String
		if !list.ReadUint16LengthPrefixed(&one) {
			return nil, fmt.Errorf("scan: sct list element")
		}
		out = append(out, append([]byte(nil), one...))
	}
	return out, nil
}

// IssuerKeyHash is the SHA-256 of the issuer's SubjectPublicKeyInfo (RFC 6962 §3.2's
// issuer_key_hash). It is captured at the handshake (the issuer sits at chain position 1) and
// stored beside the leaf in certificate_material, because an embedded SCT is signed over the
// precertificate and its leaf hash needs the issuer's key, which the leaf alone does not carry.
func IssuerKeyHash(issuerSPKI []byte) [32]byte {
	return sha256.Sum256(issuerSPKI)
}

// PrecertTBS reconstructs the tbs_certificate of the precertificate an embedded SCT was signed
// over (RFC 6962 §3.2). The precert's TBSCertificate is the FINAL certificate's TBSCertificate
// with the embedded SCT-list extension removed; the final certificate carries no poison
// extension, so nothing else changes. This is byte-surgery on the DER: it walks the
// TBSCertificate SEQUENCE, copies every field verbatim, and rewrites only the [3] EXPLICIT
// extensions element to drop the SCT-list Extension. A leaf with no extensions block, or no
// SCT-list extension, is returned with its TBS unchanged.
func PrecertTBS(leafDER []byte) ([]byte, error) {
	cert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, fmt.Errorf("scan: parse leaf for precert tbs: %w", err)
	}
	input := cryptobyte.String(cert.RawTBSCertificate)
	var tbs cryptobyte.String
	if !input.ReadASN1(&tbs, cryptobyte_asn1.SEQUENCE) {
		return nil, fmt.Errorf("scan: precert tbs sequence")
	}
	extTag := cryptobyte_asn1.Tag(3).Constructed().ContextSpecific()
	var out cryptobyte.Builder
	var buildErr error
	out.AddASN1(cryptobyte_asn1.SEQUENCE, func(child *cryptobyte.Builder) {
		for !tbs.Empty() {
			var elem cryptobyte.String
			var tag cryptobyte_asn1.Tag
			if !tbs.ReadAnyASN1Element(&elem, &tag) {
				buildErr = fmt.Errorf("scan: precert tbs element")
				return
			}
			if tag == extTag {
				if err := rewriteExtensions(child, elem, extTag); err != nil {
					buildErr = err
					return
				}
				continue
			}
			child.AddBytes(elem)
		}
	})
	if buildErr != nil {
		return nil, buildErr
	}
	return out.Bytes()
}

// rewriteExtensions copies a TBSCertificate's [3] EXPLICIT extensions element into b with the
// SCT-list Extension removed. The element is [3] { SEQUENCE OF Extension }; every Extension is
// copied verbatim except the one whose OID is the SCT list, which is dropped. Each Extension
// begins with its OID, so a cheap peek at the first OID inside the Extension SEQUENCE decides.
func rewriteExtensions(b *cryptobyte.Builder, element cryptobyte.String, extTag cryptobyte_asn1.Tag) error {
	inner := element
	var explicit cryptobyte.String
	if !inner.ReadASN1(&explicit, extTag) {
		return fmt.Errorf("scan: extensions explicit tag")
	}
	var seq cryptobyte.String
	if !explicit.ReadASN1(&seq, cryptobyte_asn1.SEQUENCE) {
		return fmt.Errorf("scan: extensions sequence")
	}
	var buildErr error
	b.AddASN1(extTag, func(e1 *cryptobyte.Builder) {
		e1.AddASN1(cryptobyte_asn1.SEQUENCE, func(e2 *cryptobyte.Builder) {
			for !seq.Empty() {
				var extElem cryptobyte.String
				if !seq.ReadASN1Element(&extElem, cryptobyte_asn1.SEQUENCE) {
					buildErr = fmt.Errorf("scan: extension element")
					return
				}
				drop, perr := extensionIsSCTList(extElem)
				if perr != nil {
					buildErr = perr
					return
				}
				if drop {
					continue
				}
				e2.AddBytes(extElem)
			}
		})
	})
	return buildErr
}

// extensionIsSCTList reports whether one DER Extension is the embedded SCT-list extension, by
// reading the OID that opens the Extension SEQUENCE. It reads a copy, so the caller's element
// is untouched and copied verbatim when kept.
func extensionIsSCTList(extElem cryptobyte.String) (bool, error) {
	var body cryptobyte.String
	if !extElem.ReadASN1(&body, cryptobyte_asn1.SEQUENCE) {
		return false, fmt.Errorf("scan: extension body")
	}
	var oid asn1.ObjectIdentifier
	if !body.ReadASN1ObjectIdentifier(&oid) {
		return false, fmt.Errorf("scan: extension oid")
	}
	return oid.Equal(oidSCTList), nil
}

// LeafHashX509 is the RFC 6962 leaf hash of an x509_entry MerkleTreeLeaf (§3.4): the leaf hash
// of a certificate whose SCT was signed over the FINAL certificate (a TLS-extension or OCSP
// SCT). The MerkleTreeLeaf is v1 / timestamped_entry / x509_entry with the leaf DER as the
// signed_entry and the SCT's extensions folded in; the leaf hash is SHA-256(0x00 ‖ leaf).
func LeafHashX509(leafDER, sctExtensions []byte, timestamp uint64) []byte {
	var b cryptobyte.Builder
	b.AddUint8(ctLeafVersionV1)
	b.AddUint8(ctLeafTypeTimestamp)
	b.AddUint64(timestamp)
	b.AddUint16(ctEntryX509)
	b.AddUint24LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(leafDER) })
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(sctExtensions) })
	return leafHash(b.BytesOrPanic())
}

// LeafHashPrecert is the RFC 6962 leaf hash of a precert_entry MerkleTreeLeaf (§3.4): the leaf
// hash of a certificate whose SCT was embedded and so signed over the precertificate. The
// signed_entry is PreCert { issuer_key_hash[32]; TBSCertificate } — the issuer key hash and
// the reconstructed precert TBS (PrecertTBS) — with the SCT's extensions folded in; the leaf
// hash is SHA-256(0x00 ‖ leaf).
func LeafHashPrecert(issuerKeyHash [32]byte, precertTBS, sctExtensions []byte, timestamp uint64) []byte {
	var b cryptobyte.Builder
	b.AddUint8(ctLeafVersionV1)
	b.AddUint8(ctLeafTypeTimestamp)
	b.AddUint64(timestamp)
	b.AddUint16(ctEntryPrecert)
	b.AddBytes(issuerKeyHash[:])
	b.AddUint24LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(precertTBS) })
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(sctExtensions) })
	return leafHash(b.BytesOrPanic())
}

// leafHash is the RFC 6962 leaf-node hash: SHA-256 of the 0x00 domain-separation prefix and the
// MerkleTreeLeaf bytes (§2.1). The 0x00 prefix distinguishes a leaf from an interior node
// (nodeHash's 0x01), so a leaf hash can never collide with an interior hash.
func leafHash(merkleTreeLeaf []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(merkleTreeLeaf)
	return h.Sum(nil)
}

// nodeHash is the RFC 6962 interior-node hash: SHA-256 of the 0x01 prefix and the two child
// hashes left‖right (§2.1).
func nodeHash(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

// --- RFC 6962 inclusion ------------------------------------------------------

// STHRoot reads the sha256_root_hash from a get-sth response body (RFC 6962 §4.3), so the
// audit-path recompute has the tree root to compare against. A body that is not the documented
// shape, or a root that is not 32 bytes, is an error the caller treats as transient (§7).
func STHRoot(body []byte) ([]byte, error) {
	var raw struct {
		RootHash string `json:"sha256_root_hash"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("scan: decode get-sth root: %w", err)
	}
	root, err := base64.StdEncoding.DecodeString(raw.RootHash)
	if err != nil {
		return nil, fmt.Errorf("scan: get-sth root base64: %w", err)
	}
	if len(root) != sha256.Size {
		return nil, fmt.Errorf("scan: get-sth root is %d bytes, want %d", len(root), sha256.Size)
	}
	return root, nil
}

// ParseProofByHash decodes a get-proof-by-hash response body (RFC 6962 §4.5) into the leaf's
// index in the tree and the audit path — the sibling hashes from the leaf to the root. A body
// that is not the documented shape, a negative index, or an audit-path node that is not 32
// bytes, is an error the caller treats as transient (§7).
func ParseProofByHash(body []byte) (index int64, auditPath [][]byte, err error) {
	var raw struct {
		LeafIndex int64    `json:"leaf_index"`
		AuditPath []string `json:"audit_path"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, nil, fmt.Errorf("scan: decode get-proof-by-hash: %w", err)
	}
	if raw.LeafIndex < 0 {
		return 0, nil, fmt.Errorf("scan: get-proof-by-hash negative index %d", raw.LeafIndex)
	}
	path := make([][]byte, 0, len(raw.AuditPath))
	for i, node := range raw.AuditPath {
		h, derr := base64.StdEncoding.DecodeString(node)
		if derr != nil {
			return 0, nil, fmt.Errorf("scan: audit path %d base64: %w", i, derr)
		}
		if len(h) != sha256.Size {
			return 0, nil, fmt.Errorf("scan: audit path %d is %d bytes, want %d", i, len(h), sha256.Size)
		}
		path = append(path, h)
	}
	return raw.LeafIndex, path, nil
}

// VerifyInclusion reports whether an audit path proves a leaf hash is at index in a tree of the
// given size whose root is root (RFC 6962 §2.1.1). It recomputes the root from the leaf hash
// and the sibling hashes and compares; a wrong index, a wrong-length path, or a mismatched root
// all report false. This is the Trillian decomposition — the path splits into the inner nodes
// (the leaf's own subtree) and the right-border nodes — and never trusts the log: the root it
// compares against is the one the caller read from the signed head, so a log that returns a
// proof for a hash it does not hold produces a root that does not match.
func VerifyInclusion(leafHash []byte, index, size int64, auditPath [][]byte, root []byte) bool {
	computed, ok := rootFromInclusionProof(leafHash, index, size, auditPath)
	if !ok {
		return false
	}
	return bytesEqual(computed, root)
}

// rootFromInclusionProof recomputes a tree root from a leaf hash and its audit path (RFC 6962
// §2.1.1, the Trillian decomposition). inner is the number of proof nodes below the point where
// the leaf's path meets the tree's right edge; border is the number above it. The inner nodes
// are combined left/right by the bits of the index; the border nodes are all combined on the
// right. It reports false when the path length does not match the decomposition — a proof of
// the wrong length for this (index, size) is never accepted.
func rootFromInclusionProof(leafHash []byte, index, size int64, auditPath [][]byte) ([]byte, bool) {
	if index < 0 || size < 0 || index >= size {
		return nil, false
	}
	inner := bits.Len64(uint64(index ^ (size - 1)))
	border := bits.OnesCount64(uint64(index >> inner))
	if len(auditPath) != inner+border {
		return nil, false
	}
	h := append([]byte(nil), leafHash...)
	for i := 0; i < inner; i++ {
		if (index>>uint(i))&1 == 0 {
			h = nodeHash(h, auditPath[i])
		} else {
			h = nodeHash(auditPath[i], h)
		}
	}
	for i := inner; i < inner+border; i++ {
		h = nodeHash(auditPath[i], h)
	}
	return h, true
}

// CheckpointRoot reads the root hash from a static-ct-api checkpoint body (C2SP signed-note):
// the origin line, the decimal tree size, then the base64 root hash on line 3 (§4.2). A body
// with fewer than three lines, or a root that is not 32 bytes, is an error the caller treats as
// transient (§7). The tree size is read by ParseCheckpoint; this reads the root beside it.
func CheckpointRoot(body []byte) ([]byte, error) {
	lines := strings.Split(string(body), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("scan: checkpoint has %d lines, want at least 3", len(lines))
	}
	root, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[2]))
	if err != nil {
		return nil, fmt.Errorf("scan: checkpoint root base64: %w", err)
	}
	if len(root) != sha256.Size {
		return nil, fmt.Errorf("scan: checkpoint root is %d bytes, want %d", len(root), sha256.Size)
	}
	return root, nil
}

// HashTilePath renders a level-0 hash-tile request path for the tile covering entry index. A
// static-ct-api log serves the Merkle leaf hashes as `tile/0/<path>` files, each up to
// CTTileWidth (256) 32-byte hashes, so the tile index is index/256 and the leaf's position in
// the tile is index%256. The path segments reuse DataTilePath's base-1000 encoding; the caller
// adds the monitoring prefix and the `.p/<W>` suffix for a partial head tile.
func HashTilePath(index int64) string {
	return "0/" + DataTilePath(index/CTTileWidth)
}

func ParseHashTile(body []byte) ([][]byte, error) {
	if len(body)%sha256.Size != 0 {
		return nil, fmt.Errorf("scan: hash tile is %d bytes, not a multiple of %d", len(body), sha256.Size)
	}
	n := len(body) / sha256.Size
	out := make([][]byte, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, body[i*sha256.Size:(i+1)*sha256.Size])
	}
	return out, nil
}

// LeafHashInTile reports whether the leaf hash sits at position index within a level-0 hash
// tile (§5, tiled path). The tile covers entries [256*floor(index/256), …); the leaf's slot is
// index%256. It reports (matches, present): present is false when the tile is too short to hold
// the slot — a head tile that has not yet grown to the SCT's index, which the caller treats as
// not-yet-inclusion rather than a mismatch.
func LeafHashInTile(leafHash []byte, index int64, tileHashes [][]byte) (matches, present bool) {
	slot := int(index % CTTileWidth)
	if slot >= len(tileHashes) {
		return false, false
	}
	return bytesEqual(leafHash, tileHashes[slot]), true
}

// AllLogs is every CT log in the embedded log_list.json, of both client kinds, regardless of
// state or temporal interval. Verification looks a log up by the id an SCT names, and an SCT
// may name a RETIRED or past-temporal shard that SelectTailLogs (which follows only currently
// readable, currently covering logs) would drop — a retired log is still readable for a
// point-check even when the tail no longer follows it. So verification selects from the whole
// list, not the tail's followed set.
func AllLogs() ([]CTLog, error) {
	var ll logList
	if err := json.Unmarshal(embeddedLogList, &ll); err != nil {
		return nil, fmt.Errorf("scan: decode ct log list: %w", err)
	}
	var out []CTLog
	for _, op := range ll.Operators {
		for _, e := range op.Logs {
			if e.LogID == "" || e.URL == "" {
				continue
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.URL, Description: e.Description})
		}
		for _, e := range op.TiledLogs {
			if e.LogID == "" || e.MonitoringURL == "" {
				continue
			}
			out = append(out, CTLog{LogID: e.LogID, URL: e.MonitoringURL, Description: e.Description, Tiled: true})
		}
	}
	return out, nil
}

// FindLogByLogID returns the CT log an SCT names, matched by the base64 of the SCT's 32-byte
// log id against each log's log_list.json log_id. It reports false when no log matches — an SCT
// from a log the embedded list does not carry, which verification treats as unverifiable rather
// than not-logged (the certificate may well be logged where we cannot check).
func FindLogByLogID(logs []CTLog, logID [32]byte) (CTLog, bool) {
	want := base64.StdEncoding.EncodeToString(logID[:])
	for _, lg := range logs {
		if lg.LogID == want {
			return lg, true
		}
	}
	return CTLog{}, false
}

// bytesEqual is a length-checked byte comparison for the hash compares above; it avoids a
// dependency on bytes for this one use and reads at the call sites as an inclusion check.
func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
