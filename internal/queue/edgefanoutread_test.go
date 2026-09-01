package queue

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
)

// certWithSANs builds one self-signed leaf carrying the given dNSName SANs and returns
// its DER. The fan-out reduction reads a certificate off the wire, so the fixture is a
// real certificate rather than a hand-written SAN list.
//
// It takes a testing.TB rather than a *testing.T so BenchmarkToEdgeFanout builds its
// fixture through the same helper the tests use (#1014). A hand-written byte slice
// leaves x509.ParseCertificate nothing to do, and would measure nothing.
func certWithSANs(t testing.TB, sans ...string) []byte {
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

// edgeFixture is one posed stored measurement: an address, the leaf's outcome, and the
// DER of the certificate that edge presented. A nil der poses a row carrying NO
// fingerprint, which is what each of the three negative outcomes stores.
type edgeFixture struct {
	addr    string
	outcome string
	der     []byte
}

// edge poses one measurement fixture.
func edge(addr, outcome string, der []byte) edgeFixture {
	return edgeFixture{addr: addr, outcome: outcome, der: der}
}

// posed splits the fixtures into the TWO reads ReadEdgeFanout issues since #1035: one
// measurement row per address, naming its certificate by fingerprint alone, and one
// material entry per DISTINCT certificate.
//
// Two fixtures carrying the same DER share a fingerprint, and the material map then
// holds ONE entry for both. That is the shared-edge shape — many addresses behind one
// CDN edge presenting one certificate — and it is what the reduction must now derive
// once rather than once per address.
//
// The fingerprint is minted with connectoutcome.Fingerprint, the same function the leaf
// keys the side store with (edgefanout.presentedMaterial), so no fixture can name a
// certificate under a key production would never mint.
func posed(fx ...edgeFixture) ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
	rows := make([]db.ListEdgeFanoutMeasurementsRow, 0, len(fx))
	material := map[string][]byte{}
	for _, f := range fx {
		r := db.ListEdgeFanoutMeasurementsRow{Address: f.addr, Outcome: f.outcome}
		if len(f.der) > 0 {
			fp := co.Fingerprint(f.der)
			r.Fingerprint = pgtype.Text{String: fp, Valid: true}
			material[fp] = f.der
		}
		rows = append(rows, r)
	}
	return rows, material
}

// reduce runs the pure reduction over posed fixtures.
func reduce(completed bool, fx ...edgeFixture) custody.EdgeFanout {
	rows, material := posed(fx...)
	return toEdgeFanout(completed, rows, material)
}

// An edge presenting identities at or above the threshold is shared; one below it is
// not. The reduction and the threshold are #984's; what this pins is that the read path
// hands the derivation the same boolean.
func TestToEdgeFanoutReadsTheSANSetOffTheCertificate(t *testing.T) {
	shared := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)
	dedicated := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold-1)...)

	got := reduce(false,
		edge("104.16.132.229", string(edgefanout.Presented), shared),
		edge("93.184.216.34", string(edgefanout.Presented), dedicated),
	)

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
//
// A negative stores a NULL FINGERPRINT (#1035), where it used to arrive as a NULL `der`
// off the join. Both spellings are a VALUE, not the absence the veto holds on: only a
// missing ROW is that.
func TestToEdgeFanoutMeasuresEveryNegativeOutcomeAsNotShared(t *testing.T) {
	for _, outcome := range []edgefanout.Outcome{edgefanout.TLSRefused, edgefanout.NoTLS, edgefanout.Unreachable} {
		rows, material := posed(edge("104.16.132.229", string(outcome), nil))
		if rows[0].Fingerprint.Valid {
			t.Fatalf("outcome %s posed a fingerprint — a negative measured no identity to name", outcome)
		}
		got := toEdgeFanout(false, rows, material)
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
	got := toEdgeFanout(false, nil, nil)
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
//
// #1035 splits that state into three spellings, and all three must reach:
//
//   - the row names NO fingerprint, which the old join rendered as a NULL `der`;
//   - the row NAMES a fingerprint the material read returned nothing for, which is the
//     shape the second read makes newly expressible;
//   - the material landed and does not decode.
func TestToEdgeFanoutReachesAPresentedRowWithNoMaterial(t *testing.T) {
	addr := netip.MustParseAddr("104.16.132.229")
	orphan := db.ListEdgeFanoutMeasurementsRow{
		Address:     addr.String(),
		Outcome:     string(edgefanout.Presented),
		Fingerprint: pgtype.Text{String: co.Fingerprint([]byte("a certificate nothing captured")), Valid: true},
	}
	cases := map[string]func() ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte){
		"no fingerprint on the row": func() ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
			return posed(edge(addr.String(), string(edgefanout.Presented), nil))
		},
		"a fingerprint the material read did not return": func() ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
			return []db.ListEdgeFanoutMeasurementsRow{orphan}, nil
		},
		"material that does not decode": func() ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
			return posed(edge(addr.String(), string(edgefanout.Presented), []byte("not a certificate")))
		},
	}
	for name, pose := range cases {
		t.Run(name, func(t *testing.T) {
			rows, material := pose()
			got := toEdgeFanout(false, rows, material)
			shared, measured := got.Shared[addr]
			if !measured || shared {
				t.Fatalf("measured=%v shared=%v, want measured=true shared=false", measured, shared)
			}
		})
	}
}

// The map is keyed on the address, never on a spelling, so the derivation's Unmap'ed
// lookup finds a row a mapped rendering wrote.
func TestToEdgeFanoutKeysOnTheUnmappedAddress(t *testing.T) {
	got := reduce(false,
		edge("::ffff:104.16.132.229", string(edgefanout.Presented), certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)),
	)
	if !got.Shared[netip.MustParseAddr("104.16.132.229")] {
		t.Fatalf("Shared = %v, want a key on the unmapped address", got.Shared)
	}
}

// A row whose address does not parse is SKIPPED, leaving no key. The address is then
// held rather than reached: a row nothing can read is a measurement nothing made, and
// withholding the probe is the safe direction.
func TestToEdgeFanoutSkipsAnUnparseableAddress(t *testing.T) {
	got := reduce(false, edge("not-an-address", string(edgefanout.Presented), nil))
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty", got.Shared)
	}
}

// THE VERDICT IS DERIVED ONCE PER DISTINCT FINGERPRINT (#1035). Many addresses behind
// ONE shared CDN edge present ONE certificate, and #1014 measured the old reduction
// parsing that certificate — and walking its several hundred SANs — once per address.
//
// The property is measured in ALLOCATIONS, because that is where the repeated parse
// lands and it is the column #1014 read the headroom off. The same row count over one
// certificate and over N distinct ones must not cost the same any more. The margin is
// a full order of magnitude, so the test pins the shape of the fix and not a Go
// release's allocation accounting.
func TestToEdgeFanoutDerivesTheVerdictOncePerDistinctFingerprint(t *testing.T) {
	const rows = 200
	sans := distinctSANs(400)

	one := certWithSANs(t, sans...)
	sharedFx := make([]edgeFixture, 0, rows)
	uniqueFx := make([]edgeFixture, 0, rows)
	for i := range rows {
		addr := fmt.Sprintf("10.0.%d.%d", i/256, i%256)
		sharedFx = append(sharedFx, edge(addr, string(edgefanout.Presented), one))
		uniqueFx = append(uniqueFx, edge(addr, string(edgefanout.Presented), certWithSANs(t, sans...)))
	}

	sharedRows, sharedMaterial := posed(sharedFx...)
	uniqueRows, uniqueMaterial := posed(uniqueFx...)
	if len(sharedMaterial) != 1 {
		t.Fatalf("the shared fixture holds %d certificates, want 1", len(sharedMaterial))
	}
	if len(uniqueMaterial) != rows {
		t.Fatalf("the unique fixture holds %d certificates, want %d", len(uniqueMaterial), rows)
	}

	sharedAllocs := testing.AllocsPerRun(2, func() {
		benchEdgeFanoutSink = toEdgeFanout(true, sharedRows, sharedMaterial)
	})
	uniqueAllocs := testing.AllocsPerRun(2, func() {
		benchEdgeFanoutSink = toEdgeFanout(true, uniqueRows, uniqueMaterial)
	})
	if sharedAllocs*10 >= uniqueAllocs {
		t.Fatalf("one certificate over %d rows allocated %.0f, %d certificates allocated %.0f: "+
			"the reduction is still keying on the address rather than on the fingerprint",
			rows, sharedAllocs, rows, uniqueAllocs)
	}
}

// The memo keys on the FINGERPRINT, so one certificate's verdict never stands in for
// another's. Two edges with opposite verdicts must each keep their own, however many
// addresses present either.
func TestToEdgeFanoutKeepsEachFingerprintsOwnVerdict(t *testing.T) {
	shared := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)
	dedicated := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold-1)...)

	got := reduce(true,
		edge("104.16.132.229", string(edgefanout.Presented), shared),
		edge("104.16.132.230", string(edgefanout.Presented), dedicated),
		edge("104.16.132.231", string(edgefanout.Presented), shared),
		edge("104.16.132.232", string(edgefanout.Presented), dedicated),
	)

	want := map[string]bool{
		"104.16.132.229": true,
		"104.16.132.230": false,
		"104.16.132.231": true,
		"104.16.132.232": false,
	}
	for addr, w := range want {
		if got.Shared[netip.MustParseAddr(addr)] != w {
			t.Errorf("%s reduced to shared=%v, want %v", addr, got.Shared[netip.MustParseAddr(addr)], w)
		}
	}
}

// The material read is asked for each fingerprint ONCE, in first-seen order, and a NULL
// one is not asked for at all. Sending the raw column would pull one DER per address,
// which is the wire cost #1035 removes.
func TestEdgeFanoutFingerprintsAsksForEachCertificateOnce(t *testing.T) {
	rows := []db.ListEdgeFanoutMeasurementsRow{
		{Address: "10.0.0.1", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "sha256:aa", Valid: true}},
		{Address: "10.0.0.2", Outcome: string(edgefanout.Unreachable)},
		{Address: "10.0.0.3", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "sha256:bb", Valid: true}},
		{Address: "10.0.0.4", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "sha256:aa", Valid: true}},
		{Address: "10.0.0.5", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "", Valid: true}},
	}

	got := edgeFanoutFingerprints(rows)
	want := []string{"sha256:aa", "sha256:bb"}
	if len(got) != len(want) {
		t.Fatalf("fingerprints = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fingerprints[%d] = %q, want %q", i, got[i], want[i])
		}
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

// The completion read is CARRIED OUT rather than acted on here. It is the ERRORED half
// of ADR-0129's fourth absence case, and since #1018 the floor is read PER LIMB — over
// the extension candidates, which this package does not hold. So the read path reports
// what it saw and custody.Estate.WithEdgeFanout decides.
//
// A Scan that RUNS and records nothing must not read as disabled here. It is enabled,
// and flattening the two would lose the disposition the declaration limb's census reads
// (custody.Estate.AddressScopeCensus).
func TestToEdgeFanoutCarriesTheCompletionOutToTheAssembler(t *testing.T) {
	for _, completed := range []bool{true, false} {
		got := toEdgeFanout(completed, nil, nil)
		if !got.Enabled {
			t.Fatalf("completed=%v: Enabled = false — the caller reads the Scan row, not this", completed)
		}
		if got.BatchCompleted != completed {
			t.Fatalf("completed=%v: BatchCompleted = %v, want %v", completed, got.BatchCompleted, completed)
		}
		if got.ExtensionErrored {
			t.Fatalf("completed=%v: ExtensionErrored = true — this package holds no candidate set and may not resolve the floor", completed)
		}
		if len(got.Shared) != 0 {
			t.Fatalf("completed=%v: Shared = %v, want empty", completed, got.Shared)
		}
	}
}

// A DECLARATION-LIMB ROW IS AN ORDINARY KEY. The recording half is blind to which limb
// a row came from, so the read path hands the derivation every measured address and the
// estate decides which of them are the gating limb's (#988, #1018).
func TestToEdgeFanoutCarriesEveryMeasuredAddressWhicheverLimbItCameFrom(t *testing.T) {
	got := reduce(true, edge("198.51.100.7", string(edgefanout.Unreachable), nil))
	if !got.Enabled || !got.BatchCompleted {
		t.Fatalf("Enabled = %v, BatchCompleted = %v, want both true", got.Enabled, got.BatchCompleted)
	}
	if _, measured := got.Shared[netip.MustParseAddr("198.51.100.7")]; !measured {
		t.Fatal("the recorded measurement did not reach the derivation")
	}
}

// fakeEdgeFanoutStore is the narrow EdgeFanoutStore posed by hand, so the READ's own
// shape — how many reads it issues, over what, and what it does with a failure — is
// tested without a database.
type fakeEdgeFanoutStore struct {
	scan        db.Scan
	rows        []db.ListEdgeFanoutMeasurementsRow
	material    []db.ListCertificateMaterialDERRow
	materialErr error
	completed   bool

	// materialAsked records every fingerprint set the material read was called with.
	// It is a slice of calls, not a set, so a test can pin that an EMPTY set is not
	// asked for at all rather than asked for and answered with nothing.
	materialAsked [][]string
}

func (f *fakeEdgeFanoutStore) GetScanByKind(context.Context, string) (db.Scan, error) {
	return f.scan, nil
}

func (f *fakeEdgeFanoutStore) ListEdgeFanoutMeasurements(context.Context) ([]db.ListEdgeFanoutMeasurementsRow, error) {
	return f.rows, nil
}

func (f *fakeEdgeFanoutStore) ListCertificateMaterialDER(_ context.Context, fingerprints []string) ([]db.ListCertificateMaterialDERRow, error) {
	f.materialAsked = append(f.materialAsked, fingerprints)
	if f.materialErr != nil {
		return nil, f.materialErr
	}
	return f.material, nil
}

func (f *fakeEdgeFanoutStore) ScanHasCompletedBatch(context.Context, string) (bool, error) {
	return f.completed, nil
}

// The read asks for each DISTINCT fingerprint once and reduces the certificate once,
// however many addresses present it. This is #1035's whole claim, read at the level of
// the store calls rather than of the reduction.
func TestReadEdgeFanoutReadsEachDistinctCertificateOnce(t *testing.T) {
	der := certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)
	fp := co.Fingerprint(der)
	f := &fakeEdgeFanoutStore{
		scan:      db.Scan{Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			{Address: "104.16.132.229", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: fp, Valid: true}},
			{Address: "104.16.132.230", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: fp, Valid: true}},
			{Address: "104.16.132.231", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: fp, Valid: true}},
		},
		material: []db.ListCertificateMaterialDERRow{{Fingerprint: fp, Der: der}},
	}

	got, err := ReadEdgeFanout(context.Background(), f)
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if len(f.materialAsked) != 1 {
		t.Fatalf("the material read ran %d times, want 1", len(f.materialAsked))
	}
	if len(f.materialAsked[0]) != 1 || f.materialAsked[0][0] != fp {
		t.Fatalf("the material read asked for %v, want the one distinct fingerprint %q", f.materialAsked[0], fp)
	}
	for _, addr := range []string{"104.16.132.229", "104.16.132.230", "104.16.132.231"} {
		if !got.Shared[netip.MustParseAddr(addr)] {
			t.Errorf("%s did not read as shared", addr)
		}
	}
}

// Where no row names a certificate, the material read is SKIPPED. An install the Scan
// has not run for, and one measuring nothing but negatives, are the ordinary cases, and
// neither should cost a round trip.
func TestReadEdgeFanoutSkipsTheMaterialReadWhereNoRowNamesACertificate(t *testing.T) {
	f := &fakeEdgeFanoutStore{
		scan:      db.Scan{Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			{Address: "198.51.100.7", Outcome: string(edgefanout.Unreachable)},
		},
	}

	got, err := ReadEdgeFanout(context.Background(), f)
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if len(f.materialAsked) != 0 {
		t.Fatalf("the material read ran %d times over %v, want not at all", len(f.materialAsked), f.materialAsked)
	}
	if shared, measured := got.Shared[netip.MustParseAddr("198.51.100.7")]; !measured || shared {
		t.Fatalf("measured=%v shared=%v, want measured=true shared=false", measured, shared)
	}
}

// A material read that FAILS returns the error, never an open reach. A failure there is
// a failure to read the certificates, not a finding that those edges present no
// identity — and reading it as the latter would reach every measured address at once,
// on the one signal that says nothing could be read.
func TestReadEdgeFanoutReturnsTheMaterialReadsFailure(t *testing.T) {
	boom := errors.New("connection reset")
	f := &fakeEdgeFanoutStore{
		scan:      db.Scan{Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			{Address: "104.16.132.229", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "sha256:aa", Valid: true}},
		},
		materialErr: boom,
	}

	got, err := ReadEdgeFanout(context.Background(), f)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the material read's own failure", err)
	}
	if got.Enabled || len(got.Shared) != 0 {
		t.Fatalf("got = %+v, want the zero record — a failed read decides nothing", got)
	}
}

// A DISABLED Scan reads nothing at all. The measurement narrows the reach only where
// the Scan is in force, so neither the measurements nor the material are pulled.
func TestReadEdgeFanoutReadsNoMaterialWhereTheScanIsNotInForce(t *testing.T) {
	f := &fakeEdgeFanoutStore{scan: db.Scan{Kind: scan.EdgeFanoutKind, Enabled: false}}

	got, err := ReadEdgeFanout(context.Background(), f)
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if got.Enabled {
		t.Fatal("Enabled = true over a disabled Scan")
	}
	if len(f.materialAsked) != 0 {
		t.Fatalf("the material read ran over a disabled Scan: %v", f.materialAsked)
	}
}
