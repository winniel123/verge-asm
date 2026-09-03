// CT verification is a stateless point-check that one certificate is logged
// (ct-source-replacement.md §5). It never enumerates, always starts from an SCT
// rather than a name (§5.1), and verifies no log signature (§4.4).
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

var oidSCTList = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}

// The log's signature is read past, never checked: inclusion recomputes to the head's root (§4.4).

type SCT struct {
	Version    uint8
	LogID      [32]byte
	Timestamp  uint64
	Extensions []byte
}

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
	// The digitally-signed prefix is two algorithm bytes before the signature (RFC 6962 §3.2).
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

const ctSCTVersionV1 = 0

func SCTLeafIndex(extensions []byte) (int64, bool) {
	// Only a tiled log's SCT carries a leaf_index, so its presence discriminates the client (§4.3).
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

func EmbeddedSCTs(leafDER []byte) ([][]byte, error) {
	// Many certificates carry SCTs only out of band, so an absent extension is not an error (§3.3).
	cert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, fmt.Errorf("scan: parse leaf for embedded scts: %w", err)
	}
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(oidSCTList) {
			continue
		}
		var inner cryptobyte.String
		outer := cryptobyte.String(ext.Value)
		// The extnValue is an OCTET STRING wrapping the SCT list, so it unwraps twice (RFC 6962 §3.3).
		if !outer.ReadASN1(&inner, cryptobyte_asn1.OCTET_STRING) {
			return nil, fmt.Errorf("scan: embedded sct octet string")
		}
		return parseSCTList(inner)
	}
	return nil, nil
}

func OCSPSCTs(ocspResponse []byte) ([][]byte, error) {
	// An OCSP or TLS-extension SCT signs the final certificate, never the precert (RFC 6962 §3.3).
	if len(ocspResponse) == 0 {
		return nil, nil
	}
	resp, err := ocsp.ParseResponse(ocspResponse, nil)
	if err != nil {
		return nil, nil // a malformed staple narrows the SCTs, never fails the verification
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

func IssuerKeyHash(issuerSPKI []byte) [32]byte {
	// The issuer SPKI is captured at the handshake because the leaf alone cannot supply it (§5.3).
	return sha256.Sum256(issuerSPKI)
}

func PrecertTBS(leafDER []byte) ([]byte, error) {
	// An embedded SCT signs the precert, whose TBS is this TBS minus the SCT-list extension (§3.2).
	cert, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil, fmt.Errorf("scan: parse leaf for precert tbs: %w", err)
	}
	input := cryptobyte.String(cert.RawTBSCertificate)
	var tbs cryptobyte.String
	if !input.ReadASN1(&tbs, cryptobyte_asn1.SEQUENCE) {
		return nil, fmt.Errorf("scan: precert tbs sequence")
	}
	// The final certificate carries no poison extension, so removing the SCT list is the only change.
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

func leafHash(merkleTreeLeaf []byte) []byte {
	// The 0x00 prefix separates domains, so a leaf hash never collides with an interior one (§2.1).
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(merkleTreeLeaf)
	return h.Sum(nil)
}

func nodeHash(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	return h.Sum(nil)
}

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

func VerifyInclusion(leafHash []byte, index, size int64, auditPath [][]byte, root []byte) bool {
	// The root comes from the caller's signed head, so a log cannot prove a hash it lacks (§2.1.1).
	computed, ok := rootFromInclusionProof(leafHash, index, size, auditPath)
	if !ok {
		return false
	}
	return bytesEqual(computed, root)
}

func rootFromInclusionProof(leafHash []byte, index, size int64, auditPath [][]byte) ([]byte, bool) {
	// This is Trillian's inner/border decomposition of the audit path (RFC 6962 §2.1.1).
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

func HashTilePath(index int64) string {
	// Level 0 is the leaf-hash tile level, and a tile holds up to 256 hashes (static-ct-api).
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

func LeafHashInTile(leafHash []byte, index int64, tileHashes [][]byte) (matches, present bool) {
	// A short head tile has not yet reached the index: not-yet-inclusion, never a mismatch (§5).
	slot := int(index % CTTileWidth)
	if slot >= len(tileHashes) {
		return false, false
	}
	return bytesEqual(leafHash, tileHashes[slot]), true
}

// A retired shard still answers a point-check, so verification applies no state filter (§5.1).

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

func FindLogByLogID(logs []CTLog, logID [32]byte) (CTLog, bool) {
	// An SCT from a log we do not carry is unverifiable, never evidence of not-logged (§5.4).
	want := base64.StdEncoding.EncodeToString(logID[:])
	for _, lg := range logs {
		if lg.LogID == want {
			return lg, true
		}
	}
	return CTLog{}, false
}

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
