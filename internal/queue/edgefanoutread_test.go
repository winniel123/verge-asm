package queue

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net/netip"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
)

// certWithSANs builds one self-signed leaf carrying the given dNSName SANs and returns
// its DER. The fan-out reduction reads a certificate off the wire, so the fixture is a
// real certificate rather than a hand-written SAN list.
func certWithSANs(t *testing.T, sans ...string) []byte {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "edge"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(1<<31, 0),
		DNSNames:     sans,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, key.Public(), key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	return der
}

// distinctSANs returns n SANs on n distinct registrable domains, so the fan-out count
// equals n exactly.
func distinctSANs(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("www.tenant-%d.example", i))
	}
	return out
}

func row(addr, outcome string, der []byte) db.ListEdgeFanoutMeasurementsRow {
	return db.ListEdgeFanoutMeasurementsRow{Address: addr, Outcome: outcome, Der: der}
}

// An edge presenting identities at or above the threshold is shared; one below it is
// not. The reduction and the threshold are #984's; what this pins is that the read path
// hands the derivation the same boolean.
func TestToEdgeFanoutReadsTheSANSetOffTheCertificate(t *testing.T) {
	shared := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)
	dedicated := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold-1)...)

	got := toEdgeFanout(false, []db.ListEdgeFanoutMeasurementsRow{
		row("104.16.132.229", string(edgefanout.Presented), shared),
		row("93.184.216.34", string(edgefanout.Presented), dedicated),
	})

	if !got.Enabled {
		t.Fatal("Enabled = false, want true — the caller reads the Scan row, not this")
	}
	if !got.Shared[netip.MustParseAddr("104.16.132.229")] {
		t.Fatalf("an edge presenting %d registrable domains did not read as shared", custody.SharedEdgeThreshold)
	}
	if got.Shared[netip.MustParseAddr("93.184.216.34")] {
		t.Fatalf("an edge presenting %d registrable domains read as shared", custody.SharedEdgeThreshold-1)
	}
}

// Each negative outcome MEASURED the address and found no identity there. That reduces
// to a fan-out of zero — measured and not-shared — so the address is reached, never
// held. A held address would be withheld forever on an edge that simply refuses TLS.
func TestToEdgeFanoutMeasuresEveryNegativeOutcomeAsNotShared(t *testing.T) {
	for _, outcome := range []edgefanout.Outcome{edgefanout.TLSRefused, edgefanout.NoTLS, edgefanout.Unreachable} {
		got := toEdgeFanout(false, []db.ListEdgeFanoutMeasurementsRow{row("104.16.132.229", string(outcome), nil)})
		shared, measured := got.Shared[netip.MustParseAddr("104.16.132.229")]
		if !measured {
			t.Fatalf("outcome %s left the address unmeasured — a negative is a value, not an absence", outcome)
		}
		if shared {
			t.Fatalf("outcome %s read as shared", outcome)
		}
	}
}

// An address the Scan did not measure carries no row, so it gets NO KEY. That missing
// key is what the derivation reads as measurement pending and holds on.
func TestToEdgeFanoutGivesAnUnmeasuredAddressNoKey(t *testing.T) {
	got := toEdgeFanout(false, nil)
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty — an absence is never a value", got.Shared)
	}
	if _, measured := got.Shared[netip.MustParseAddr("104.16.132.229")]; measured {
		t.Fatal("an unmeasured address carried a key")
	}
}

// A `presented` row whose certificate material never landed reduces to a fan-out of
// zero and is REACHED, not held. Holding it would withhold the address forever on an
// error that never clears, and a silently missing estate is the direction ADR-0129
// refuses; probing one edge too many is the loud direction it accepts.
func TestToEdgeFanoutReachesAPresentedRowWithNoMaterial(t *testing.T) {
	for _, der := range [][]byte{nil, []byte("not a certificate")} {
		got := toEdgeFanout(false, []db.ListEdgeFanoutMeasurementsRow{row("104.16.132.229", string(edgefanout.Presented), der)})
		shared, measured := got.Shared[netip.MustParseAddr("104.16.132.229")]
		if !measured || shared {
			t.Fatalf("der %q: measured=%v shared=%v, want measured=true shared=false", der, measured, shared)
		}
	}
}

// The map is keyed on the address, never on a spelling, so the derivation's Unmap'ed
// lookup finds a row a mapped rendering wrote.
func TestToEdgeFanoutKeysOnTheUnmappedAddress(t *testing.T) {
	got := toEdgeFanout(false, []db.ListEdgeFanoutMeasurementsRow{
		row("::ffff:104.16.132.229", string(edgefanout.Presented), certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)),
	})
	if !got.Shared[netip.MustParseAddr("104.16.132.229")] {
		t.Fatalf("Shared = %v, want a key on the unmapped address", got.Shared)
	}
}

// A row whose address does not parse is SKIPPED, leaving no key. The address is then
// held rather than reached: a row nothing can read is a measurement nothing made, and
// withholding the probe is the safe direction.
func TestToEdgeFanoutSkipsAnUnparseableAddress(t *testing.T) {
	got := toEdgeFanout(false, []db.ListEdgeFanoutMeasurementsRow{row("not-an-address", string(edgefanout.Presented), nil)})
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty", got.Shared)
	}
}

// The SAN read takes the dNSName SANs alone and folds in no subject common name. A CN
// is not a SAN, and the fixture's CN is deliberately a name no SAN repeats.
func TestEdgeFanoutSANsReadsTheDNSNamesAlone(t *testing.T) {
	der := certWithSANs(t, "www.example.com", "*.example.com")
	got := edgeFanoutSANs(der)
	want := []string{"www.example.com", "*.example.com"}
	if len(got) != len(want) {
		t.Fatalf("sans = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sans[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// An empty store on a Scan that has ALREADY COMPLETED A BATCH is an ERRORED Scan, not a
// pending one. It is enabled, it ran, and it recorded nothing — the shape a prober
// binary older than #982 produces on every tick, forever. The reach opens, which is
// ADR-0129's fourth absence case and the behaviour before the measurement existed.
//
// Without this floor hold-then-open has none: every in-zone direct-A address would be
// withheld from every Scan the gate covers, silently and for good.
func TestToEdgeFanoutOpensTheReachOnAScanThatRunsAndMeasuresNothing(t *testing.T) {
	got := toEdgeFanout(true, nil)
	if got.Enabled {
		t.Fatal("Enabled = true on a Scan that completed a Batch and recorded nothing — the veto has no floor")
	}
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty", got.Shared)
	}
}

// A Scan that has NOT run yet is pending, never errored. Its candidates are held,
// bounded by the daily cadence, so a fresh install and a newly-declared extension never
// show appear-then-withdraw churn.
func TestToEdgeFanoutHoldsAScanThatHasNotRunYet(t *testing.T) {
	got := toEdgeFanout(false, nil)
	if !got.Enabled {
		t.Fatal("Enabled = false before the Scan has run — a pending Scan must hold, not open")
	}
}

// One recorded measurement of ANY kind lifts the floor. The three negatives each record
// a row, so a run of failed handshakes leaves the Scan in force and its unmeasured
// candidates held — the floor cannot be reached by a bad network.
func TestOneMeasurementLiftsTheErroredFloor(t *testing.T) {
	got := toEdgeFanout(true, []db.ListEdgeFanoutMeasurementsRow{
		row("104.16.132.229", string(edgefanout.Unreachable), nil),
	})
	if !got.Enabled {
		t.Fatal("Enabled = false with a measurement in the store — a recorded negative is the Scan working")
	}
	if _, measured := got.Shared[netip.MustParseAddr("104.16.132.229")]; !measured {
		t.Fatal("the recorded measurement did not reach the derivation")
	}
}
