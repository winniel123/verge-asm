package scan

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"golang.org/x/crypto/cryptobyte"
)

// refMTH is the RFC 6962 §2.1 Merkle Tree Hash of a list of leaf hashes, computed by the
// straight recursive definition. It is the trusted reference the tile/proof code is checked
// against: refMTH computes the root a different way than rootFromInclusionProof reassembles it.
func refMTH(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		h := sha256.Sum256(nil)
		return h[:]
	}
	if len(leaves) == 1 {
		return leaves[0]
	}
	k := largestPowerOfTwoLessThan(len(leaves))
	return nodeHash(refMTH(leaves[:k]), refMTH(leaves[k:]))
}

// refPath is the RFC 6962 §2.1.1 audit path for the leaf at index m in a tree of the given
// leaves, computed by the straight recursive definition — the reference the parsed audit path
// is checked against.
func refPath(m int, leaves [][]byte) [][]byte {
	if len(leaves) <= 1 {
		return nil
	}
	k := largestPowerOfTwoLessThan(len(leaves))
	if m < k {
		return append(refPath(m, leaves[:k]), refMTH(leaves[k:]))
	}
	return append(refPath(m-k, leaves[k:]), refMTH(leaves[:k]))
}

func largestPowerOfTwoLessThan(n int) int {
	k := 1
	for k<<1 < n {
		k <<= 1
	}
	return k
}

func leafHashOf(b []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(b)
	return h.Sum(nil)
}

func TestVerifyInclusionAgainstReferenceTree(t *testing.T) {
	// A tree of 13 leaves exercises every right-border shape (13 = 1101, so the border has
	// several set bits). Verify every leaf at every tree size up to 13.
	all := make([][]byte, 13)
	for i := range all {
		all[i] = leafHashOf([]byte{byte(i), 0xAB})
	}
	for size := 1; size <= len(all); size++ {
		leaves := all[:size]
		root := refMTH(leaves)
		for idx := 0; idx < size; idx++ {
			path := refPath(idx, leaves)
			if !VerifyInclusion(leaves[idx], int64(idx), int64(size), path, root) {
				t.Fatalf("size=%d idx=%d: valid proof rejected", size, idx)
			}
			// A tampered root must not verify.
			bad := append([]byte(nil), root...)
			bad[0] ^= 0xFF
			if VerifyInclusion(leaves[idx], int64(idx), int64(size), path, bad) {
				t.Fatalf("size=%d idx=%d: tampered root accepted", size, idx)
			}
			// A tampered leaf hash must not verify against the true root.
			badLeaf := append([]byte(nil), leaves[idx]...)
			badLeaf[0] ^= 0xFF
			if VerifyInclusion(badLeaf, int64(idx), int64(size), path, root) {
				t.Fatalf("size=%d idx=%d: tampered leaf accepted", size, idx)
			}
		}
	}
}

func TestVerifyInclusionRejectsWrongLengthPath(t *testing.T) {
	leaves := [][]byte{leafHashOf([]byte("a")), leafHashOf([]byte("b")), leafHashOf([]byte("c"))}
	root := refMTH(leaves)
	good := refPath(1, leaves)
	// Too short and too long are both rejected — the decomposition fixes the exact length.
	if VerifyInclusion(leaves[1], 1, 3, good[:len(good)-1], root) {
		t.Fatal("short path accepted")
	}
	if VerifyInclusion(leaves[1], 1, 3, append(good, leafHashOf([]byte("x"))), root) {
		t.Fatal("long path accepted")
	}
	if VerifyInclusion(leaves[0], 5, 3, good, root) {
		t.Fatal("out-of-range index accepted")
	}
}

// TestPrecertTBSRemovesSCTList builds two otherwise-identical certificates — one WITH an
// embedded SCT-list extension, one WITHOUT — and asserts PrecertTBS on the first yields the
// exact TBSCertificate bytes of the second. This validates the DER surgery precisely: the only
// difference between the two certs is the one extension PrecertTBS must drop.
func TestPrecertTBSRemovesSCTList(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// A fixed template so both certs share every field but the SCT extension. Include a
	// second extension (BasicConstraints) so both certs carry an extensions block; without
	// one, removing the sole extension would leave an empty block the twin never had.
	base := &x509.Certificate{
		SerialNumber:          big.NewInt(0x0102030405),
		Subject:               pkix.Name{CommonName: "verify.example"},
		NotBefore:             time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:              time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		DNSNames:              []string{"verify.example"},
		BasicConstraintsValid: true,
	}
	withoutSCT := *base
	withSCT := *base
	withSCT.ExtraExtensions = []pkix.Extension{{
		Id:    oidSCTList,
		Value: []byte{0x04, 0x03, 0x01, 0x02, 0x03}, // any bytes; PrecertTBS drops the whole extension
	}}

	derWith, err := x509.CreateCertificate(rand.Reader, &withSCT, &withSCT, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	derWithout, err := x509.CreateCertificate(rand.Reader, &withoutSCT, &withoutSCT, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	got, err := PrecertTBS(derWith)
	if err != nil {
		t.Fatal(err)
	}
	certWithout, err := x509.ParseCertificate(derWithout)
	if err != nil {
		t.Fatal(err)
	}
	if !bytesEqual(got, certWithout.RawTBSCertificate) {
		t.Fatalf("reconstructed precert TBS (%d bytes) does not match the SCT-free twin (%d bytes)", len(got), len(certWithout.RawTBSCertificate))
	}
	// And the reconstruction must still be a parseable TBSCertificate with the extension gone.
	if hasSCTListExtension(t, got) {
		t.Fatal("reconstructed TBS still carries the SCT-list extension")
	}
}

func hasSCTListExtension(t *testing.T, tbs []byte) bool {
	t.Helper()
	// Wrap the TBS back into a minimal cert is heavy; instead scan for the SCT OID DER.
	oidDER, err := asn1.Marshal(oidSCTList)
	if err != nil {
		t.Fatal(err)
	}
	return indexOf(tbs, oidDER) >= 0
}

func indexOf(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if bytesEqual(haystack[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func buildSCT(logID [32]byte, timestamp uint64, extensions []byte) []byte {
	var b cryptobyte.Builder
	b.AddUint8(0) // v1
	b.AddBytes(logID[:])
	b.AddUint64(timestamp)
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes(extensions) })
	b.AddUint8(4) // hash algorithm (sha256)
	b.AddUint8(3) // signature algorithm (ecdsa)
	b.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) { c.AddBytes([]byte{0xDE, 0xAD}) })
	return b.BytesOrPanic()
}

func TestParseSCTRoundTrip(t *testing.T) {
	var logID [32]byte
	for i := range logID {
		logID[i] = byte(i)
	}
	raw := buildSCT(logID, 0x0000018FabcdefAB, nil)
	sct, err := ParseSCT(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sct.LogID != logID {
		t.Fatalf("log id = %x, want %x", sct.LogID, logID)
	}
	if sct.Timestamp != 0x0000018FabcdefAB {
		t.Fatalf("timestamp = %#x", sct.Timestamp)
	}
	if len(sct.Extensions) != 0 {
		t.Fatalf("extensions = %x, want empty", sct.Extensions)
	}
	if _, err := ParseSCT(append(raw, 0x00)); err == nil {
		t.Fatal("trailing byte accepted")
	}
}

func TestSCTLeafIndex(t *testing.T) {
	// A static-ct-api leaf_index extension: type 0, opaque16 data of 5 big-endian bytes.
	idx := int64(0x0102030405)
	var ext cryptobyte.Builder
	ext.AddUint8(0) // leaf_index type
	ext.AddUint16LengthPrefixed(func(c *cryptobyte.Builder) {
		var five [5]byte
		v := uint64(idx)
		for i := 4; i >= 0; i-- {
			five[i] = byte(v)
			v >>= 8
		}
		c.AddBytes(five[:])
	})
	got, ok := SCTLeafIndex(ext.BytesOrPanic())
	if !ok || got != idx {
		t.Fatalf("SCTLeafIndex = %d, %v; want %d, true", got, ok, idx)
	}
	if _, ok := SCTLeafIndex(nil); ok {
		t.Fatal("empty extensions reported a leaf index")
	}
}

func TestEmbeddedSCTs(t *testing.T) {
	var logID [32]byte
	logID[0] = 0x11
	sct := buildSCT(logID, 42, nil)

	var list cryptobyte.Builder
	list.AddUint16LengthPrefixed(func(outer *cryptobyte.Builder) {
		outer.AddUint16LengthPrefixed(func(one *cryptobyte.Builder) { one.AddBytes(sct) })
	})
	octet, err := asn1.Marshal(list.BytesOrPanic())
	if err != nil {
		t.Fatal(err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(9),
		Subject:      pkix.Name{CommonName: "embed.example"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		ExtraExtensions: []pkix.Extension{{
			Id:    oidSCTList,
			Value: octet,
		}},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := EmbeddedSCTs(der)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d embedded SCTs, want 1", len(got))
	}
	if !bytesEqual(got[0], sct) {
		t.Fatalf("extracted SCT does not match the embedded one")
	}
	// A cert with no SCT-list extension yields nothing, cleanly.
	plain := &x509.Certificate{SerialNumber: big.NewInt(10), Subject: pkix.Name{CommonName: "plain"}, NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	plainDER, err := x509.CreateCertificate(rand.Reader, plain, plain, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	none, err := EmbeddedSCTs(plainDER)
	if err != nil || len(none) != 0 {
		t.Fatalf("EmbeddedSCTs on a plain cert = %d, %v", len(none), err)
	}
}

func TestLeafHashX509IsLeafPrefixed(t *testing.T) {
	leafDER := []byte{0x30, 0x03, 0x02, 0x01, 0x07} // any bytes; the hash does not parse them
	ts := uint64(1700000000000)
	got := LeafHashX509(leafDER, nil, ts)

	// Rebuild the MerkleTreeLeaf independently and hash it, to pin the byte layout.
	want := make([]byte, 0, 32)
	tsb := make([]byte, 8)
	binary.BigEndian.PutUint64(tsb, ts)
	mtl := []byte{0, 0} // version v1, leaf_type timestamped_entry
	mtl = append(mtl, tsb...)
	mtl = append(mtl, 0, 0)                                                              // entry_type x509_entry
	mtl = append(mtl, byte(len(leafDER)>>16), byte(len(leafDER)>>8), byte(len(leafDER))) // opaque24 length
	mtl = append(mtl, leafDER...)
	mtl = append(mtl, 0, 0) // empty extensions
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(mtl)
	want = h.Sum(nil)
	if !bytesEqual(got, want) {
		t.Fatalf("LeafHashX509 = %x, want %x", got, want)
	}
}

func TestHashTilePathAndLeafInTile(t *testing.T) {
	if got := HashTilePath(0); got != "0/000" {
		t.Fatalf("HashTilePath(0) = %q", got)
	}
	if got := HashTilePath(CTTileWidth + 5); got != "0/001" {
		t.Fatalf("HashTilePath(261) = %q", got)
	}
	tile := [][]byte{leafHashOf([]byte("a")), leafHashOf([]byte("b")), leafHashOf([]byte("c"))}
	if m, present := LeafHashInTile(tile[1], 1, tile); !present || !m {
		t.Fatalf("slot 1 = %v, %v; want match", m, present)
	}
	if m, present := LeafHashInTile(tile[1], 2, tile); !present || m {
		t.Fatalf("slot 2 vs wrong leaf = %v, %v; want present, no-match", m, present)
	}
	// A slot within this tile's range but past the leaves it holds (a head tile not yet grown
	// to the SCT's index): present is false, so the caller treats it as not-yet-inclusion.
	if _, present := LeafHashInTile(tile[0], 5, tile); present {
		t.Fatal("slot beyond the tile's leaves reported present")
	}
}

func TestParseHashTile(t *testing.T) {
	a := sha256.Sum256([]byte("a"))
	b := sha256.Sum256([]byte("b"))
	body := append(append([]byte(nil), a[:]...), b[:]...)
	got, err := ParseHashTile(body)
	if err != nil || len(got) != 2 {
		t.Fatalf("ParseHashTile = %d, %v", len(got), err)
	}
	if _, err := ParseHashTile(body[:len(body)-1]); err == nil {
		t.Fatal("non-multiple-of-32 tile accepted")
	}
}

func TestFindLogByLogID(t *testing.T) {
	var id [32]byte
	id[0] = 0x7A
	logs := []CTLog{
		{LogID: base64.StdEncoding.EncodeToString(id[:]), URL: "https://log.example/", Description: "example"},
	}
	if lg, ok := FindLogByLogID(logs, id); !ok || lg.Description != "example" {
		t.Fatalf("FindLogByLogID = %+v, %v", lg, ok)
	}
	var other [32]byte
	other[0] = 0x01
	if _, ok := FindLogByLogID(logs, other); ok {
		t.Fatal("unknown log id matched")
	}
}

func TestAllLogsIncludesEntries(t *testing.T) {
	logs, err := AllLogs()
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) == 0 {
		t.Fatal("AllLogs returned nothing from the embedded list")
	}
}
