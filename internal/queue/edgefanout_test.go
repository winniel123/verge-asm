package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

func edgeLine(address, outcome, fingerprint string) wire.Observation {
	v := map[string]string{"outcome": outcome}
	if fingerprint != "" {
		v["fingerprint"] = fingerprint
	}
	raw, _ := json.Marshal(v)
	return wire.Observation{Batch: "b", Kind: edgefanout.Kind, Address: address, Data: raw}
}

func edgeInstant() time.Time {
	return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
}

// One row per measured address, carrying the outcome and — on `presented` alone — the
// served certificate's fingerprint. The row records the fingerprint and never the DER:
// the certificate material rides the same batch into its own side store.
func TestToEdgeFanoutRowsRecordsOneRowPerMeasuredAddress(t *testing.T) {
	at := edgeInstant()
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 42, tstz(at), []wire.Observation{
		edgeLine("93.184.216.34", "presented", "sha256:aa"),
		edgeLine("104.16.132.229", "tls-refused", ""),
	})
	if len(dropped) != 0 {
		t.Fatalf("dropped = %v, want none", dropped)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if rows[0].BatchID != 42 || rows[0].Address != "93.184.216.34" || rows[0].Outcome != "presented" {
		t.Fatalf("row 0 = %+v", rows[0])
	}
	if !rows[0].Fingerprint.Valid || rows[0].Fingerprint.String != "sha256:aa" {
		t.Fatalf("row 0 fingerprint = %+v, want sha256:aa", rows[0].Fingerprint)
	}
	if !rows[0].MeasuredAt.Valid || !rows[0].MeasuredAt.Time.Equal(at) {
		t.Fatalf("row 0 measured_at = %+v, want %s", rows[0].MeasuredAt, at)
	}
	if rows[1].Outcome != "tls-refused" || rows[1].Fingerprint.Valid {
		t.Fatalf("row 1 = %+v, want tls-refused with a NULL fingerprint", rows[1])
	}
}

// The three negatives are each a value in their own right. Every one persists, and none
// carries a fingerprint: an absence is never a value, so a candidate the Scan did not
// measure is the one that carries no row.
func TestToEdgeFanoutRowsKeepsEveryNegative(t *testing.T) {
	at := edgeInstant()
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{
		edgeLine("93.184.216.34", "tls-refused", ""),
		edgeLine("93.184.216.35", "no-tls", ""),
		edgeLine("93.184.216.36", "unreachable", ""),
	})
	if len(dropped) != 0 || len(rows) != 3 {
		t.Fatalf("rows = %d, dropped = %v, want 3 rows and no drops", len(rows), dropped)
	}
	for i, want := range []string{"tls-refused", "no-tls", "unreachable"} {
		if rows[i].Outcome != want || rows[i].Fingerprint.Valid {
			t.Fatalf("row %d = %+v, want %s with a NULL fingerprint", i, rows[i], want)
		}
	}
}

// A line of any other kind is not this store's. The completion path runs this fold on
// every batch, so a dns or connect-outcome batch must produce no row here.
func TestToEdgeFanoutRowsIgnoresOtherKinds(t *testing.T) {
	at := edgeInstant()
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{
		{Kind: "resolution-walk", Facet: "resolution", Subject: "www.example.com"},
		{Kind: "connect-outcome", Facet: "reachability", Subject: "93.184.216.34:443/tcp"},
	})
	if len(rows) != 0 || len(dropped) != 0 {
		t.Fatalf("rows = %v, dropped = %v, want neither", rows, dropped)
	}
}

// A line this leaf could not have emitted is dropped and named, never persisted and
// never guessed at. A fabricated row would feed the custody-extension veto an answer
// nothing measured.
func TestToEdgeFanoutRowsDropsWhatTheLeafCannotEmit(t *testing.T) {
	at := edgeInstant()
	cases := []struct {
		name string
		obs  wire.Observation
	}{
		{"outcome outside the union", edgeLine("93.184.216.34", "shared-edge", "")},
		{"presented with no fingerprint", edgeLine("93.184.216.34", "presented", "")},
		{"negative carrying a fingerprint", edgeLine("93.184.216.34", "no-tls", "sha256:aa")},
		{"address that does not parse", edgeLine("not-an-address", "unreachable", "")},
		{"no address at all", edgeLine("", "unreachable", "")},
		{"unparseable payload", wire.Observation{Kind: edgefanout.Kind, Address: "93.184.216.34", Data: json.RawMessage(`{`)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{c.obs})
			if len(rows) != 0 {
				t.Fatalf("rows = %+v, want none", rows)
			}
			if len(dropped) != 1 {
				t.Fatalf("dropped = %v, want one reason", dropped)
			}
		})
	}
}

// A malformed line never costs the legitimate lines in the same batch their commit: a
// compromised prober cannot turn one bad row into a queue denial of service.
func TestToEdgeFanoutRowsKeepsTheGoodLinesBesideABadOne(t *testing.T) {
	at := edgeInstant()
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{
		edgeLine("93.184.216.34", "presented", "sha256:aa"),
		edgeLine("93.184.216.35", "shared-edge", ""),
		edgeLine("93.184.216.36", "unreachable", ""),
	})
	if len(rows) != 2 || len(dropped) != 1 {
		t.Fatalf("rows = %d, dropped = %v, want 2 rows and 1 drop", len(rows), dropped)
	}
	if rows[0].Address != "93.184.216.34" || rows[1].Address != "93.184.216.36" {
		t.Fatalf("rows = %+v", rows)
	}
}

// An address repeated in one batch keeps its first row alone. The leaf handshakes each
// distinct address once, so a repeat is a line the batch had no measurement for — and
// two rows at one instant would leave the newest-per-address read a coin toss.
func TestToEdgeFanoutRowsDropsARepeatedAddress(t *testing.T) {
	at := edgeInstant()
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{
		edgeLine("93.184.216.34", "presented", "sha256:aa"),
		edgeLine("::ffff:93.184.216.34", "unreachable", ""),
	})
	if len(rows) != 1 || len(dropped) != 1 {
		t.Fatalf("rows = %d, dropped = %v, want 1 row and 1 drop", len(rows), dropped)
	}
	if rows[0].Outcome != "presented" {
		t.Fatalf("row = %+v, want the first measurement kept", rows[0])
	}
}

// The address is stored in its netip form, so a lookup never turns on a spelling. A
// mapped or padded rendering folds to the same key the dispatcher enqueued.
func TestToEdgeFanoutRowsNormalisesTheAddress(t *testing.T) {
	at := edgeInstant()
	rows, _ := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{
		edgeLine(" ::ffff:93.184.216.34 ", "unreachable", ""),
	})
	if len(rows) != 1 || rows[0].Address != "93.184.216.34" {
		t.Fatalf("rows = %+v, want the unmapped form", rows)
	}
}

// The fold admits itself on the kind the DISPATCHER enqueued, never on a line's
// self-declared kind. Without this, a prober handling a dns job — whose scope denotes
// names alone, so the address dimension has nothing to gate it against — could append
// one `edge-fanout` line and mint a measurement of any address it liked, which the veto
// (#985) would then read as an answer to a probe nothing ran.
func TestToEdgeFanoutRowsWritesNothingForAnotherJobKind(t *testing.T) {
	at := edgeInstant()
	injected := []wire.Observation{edgeLine("93.184.216.34", "presented", "sha256:aa")}
	for _, kind := range []string{scan.DNSKind, scan.HotKind, scan.ZoneKind, scan.HTTPIdentityKind, ""} {
		rows, dropped := toEdgeFanoutRows(kind, 1, tstz(at), injected)
		if len(rows) != 0 || len(dropped) != 0 {
			t.Fatalf("job kind %q wrote rows = %+v, dropped = %v, want neither", kind, rows, dropped)
		}
	}
}

// This leaf's lines carry no facet. One claiming a facet was gated on its SUBJECT
// rather than on the Address this fold reads, so its address was never authorised and
// the row must not be written.
func TestToEdgeFanoutRowsDropsALineClaimingAFacet(t *testing.T) {
	at := edgeInstant()
	line := edgeLine("93.184.216.34", "unreachable", "")
	line.Facet = "reachability"
	line.Subject = "104.16.132.229:443/tcp"
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{line})
	if len(rows) != 0 || len(dropped) != 1 {
		t.Fatalf("rows = %+v, dropped = %v, want no row and one reason", rows, dropped)
	}
}

// Where the line carries certificate material, the fingerprint the row stores and the
// key that material lands under are the same value — the side store's key is recomputed
// from its own DER, so a disagreement would leave the row naming a certificate that
// store does not hold, and #984's SAN read joining to nothing.
func TestToEdgeFanoutRowsDropsAFingerprintDisagreeingWithItsMaterial(t *testing.T) {
	at := edgeInstant()
	line := edgeLine("93.184.216.34", "presented", "sha256:aa")
	line.CertMaterial = &wire.CertMaterial{Fingerprint: "sha256:bb", DER: []byte{0x30}}
	rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{line})
	if len(rows) != 0 || len(dropped) != 1 {
		t.Fatalf("rows = %+v, dropped = %v, want no row and one reason", rows, dropped)
	}

	// The agreeing pair is written, and a presented handshake that carried a chain but
	// no DER carries no material and is not tested against one.
	line.CertMaterial = &wire.CertMaterial{Fingerprint: "sha256:aa", DER: []byte{0x30}}
	if rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{line}); len(rows) != 1 || len(dropped) != 0 {
		t.Fatalf("agreeing pair: rows = %+v, dropped = %v, want one row", rows, dropped)
	}
	line.CertMaterial = nil
	if rows, dropped := toEdgeFanoutRows(scan.EdgeFanoutKind, 1, tstz(at), []wire.Observation{line}); len(rows) != 1 || len(dropped) != 0 {
		t.Fatalf("no material: rows = %+v, dropped = %v, want one row", rows, dropped)
	}
}
