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
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	co "github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
)

func certWithSANs(t testing.TB, sans ...string) []byte {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	// A hand-written SAN list leaves x509.ParseCertificate nothing to do and would measure nothing.
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

func distinctSANs(n int) []string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, fmt.Sprintf("www.tenant-%d.example", i))
	}
	return out
}

type edgeFixture struct {
	addr    string
	outcome string
	der     []byte
}

func edge(addr, outcome string, der []byte) edgeFixture {
	return edgeFixture{addr: addr, outcome: outcome, der: der}
}

func posed(fx ...edgeFixture) ([]db.ListEdgeFanoutMeasurementsRow, map[string][]byte) {
	rows := make([]db.ListEdgeFanoutMeasurementsRow, 0, len(fx))
	material := map[string][]byte{}
	for _, f := range fx {
		r := db.ListEdgeFanoutMeasurementsRow{Address: f.addr, Outcome: f.outcome}
		if len(f.der) > 0 {
			// Production keys the side store with this function, so no fixture mints a key it never would.
			fp := co.Fingerprint(f.der)
			r.Fingerprint = pgtype.Text{String: fp, Valid: true}
			material[fp] = f.der
		}
		rows = append(rows, r)
	}
	return rows, material
}

func reduce(completed bool, fx ...edgeFixture) custody.EdgeFanout {
	rows, material := posed(fx...)
	return toEdgeFanout(completed, rows, material)
}

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

func TestToEdgeFanoutMeasuresEveryNegativeOutcomeAsNotShared(t *testing.T) {
	// A negative is a value; only a missing row is the absence that withholds a probe (ADR-0129).
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

func TestToEdgeFanoutGivesAnUnmeasuredAddressNoKey(t *testing.T) {
	got := toEdgeFanout(false, nil, nil)
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty — an absence is never a value", got.Shared)
	}
	if _, measured := got.Shared[netip.MustParseAddr("104.16.132.229")]; measured {
		t.Fatal("an unmeasured address carried a key")
	}
}

func TestToEdgeFanoutReachesAPresentedRowWithNoMaterial(t *testing.T) {
	// A silently missing estate is the direction ADR-0129 §2 refuses; one edge too many is loud.
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

func TestToEdgeFanoutKeysOnTheUnmappedAddress(t *testing.T) {
	got := reduce(false,
		edge("::ffff:104.16.132.229", string(edgefanout.Presented), certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...)),
	)
	if !got.Shared[netip.MustParseAddr("104.16.132.229")] {
		t.Fatalf("Shared = %v, want a key on the unmapped address", got.Shared)
	}
}

func TestToEdgeFanoutSkipsAnUnparseableAddress(t *testing.T) {
	got := reduce(false, edge("not-an-address", string(edgefanout.Presented), nil))
	if len(got.Shared) != 0 {
		t.Fatalf("Shared = %v, want empty", got.Shared)
	}
}

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

	// The margin is an order of magnitude, so this pins the fix's shape, not Go's accounting (#1014).
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

func TestEdgeFanoutFingerprintsAsksForEachCertificateOnce(t *testing.T) {
	// Sending the raw column instead would pull one DER per address, which is the wire cost (#1035).
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

func TestEdgeFanoutSANsReadsTheDNSNamesAlone(t *testing.T) {
	// The fixture's CN is deliberately a name no SAN repeats, so folding it in would show here.
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

func TestToEdgeFanoutCarriesTheCompletionOutToTheAssembler(t *testing.T) {
	// Flattening enabled into completed would lose the disposition the census reads (#1018).
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

func TestToEdgeFanoutCarriesEveryMeasuredAddressWhicheverLimbItCameFrom(t *testing.T) {
	got := reduce(true, edge("198.51.100.7", string(edgefanout.Unreachable), nil))
	if !got.Enabled || !got.BatchCompleted {
		t.Fatalf("Enabled = %v, BatchCompleted = %v, want both true", got.Enabled, got.BatchCompleted)
	}
	if _, measured := got.Shared[netip.MustParseAddr("198.51.100.7")]; !measured {
		t.Fatal("the recorded measurement did not reach the derivation")
	}
}

type fakeEdgeFanoutStore struct {
	scan        db.Scan
	rows        []db.ListEdgeFanoutMeasurementsRow
	material    []db.ListCertificateMaterialDERRow
	materialErr error
	completed   bool

	materialAsked [][]string

	boundsAsked  [][]string
	unboundReads int
	rowsErr      error
}

func (f *fakeEdgeFanoutStore) GetScanByKind(context.Context, string) (db.Scan, error) {
	return f.scan, nil
}

func (f *fakeEdgeFanoutStore) ListEdgeFanoutMeasurements(context.Context) ([]db.ListEdgeFanoutMeasurementsRow, error) {
	f.unboundReads++
	if f.rowsErr != nil {
		return nil, f.rowsErr
	}
	return f.rows, nil
}

func (f *fakeEdgeFanoutStore) ListEdgeFanoutMeasurementsOver(_ context.Context, addresses []string) ([]db.ListEdgeFanoutMeasurementsOverRow, error) {
	// The fake mirrors the SQL's address = ANY(...), or a dropped row would not show here (#1036).
	f.boundsAsked = append(f.boundsAsked, addresses)
	if f.rowsErr != nil {
		return nil, f.rowsErr
	}
	want := make(map[string]struct{}, len(addresses))
	for _, a := range addresses {
		want[a] = struct{}{}
	}
	out := []db.ListEdgeFanoutMeasurementsOverRow{}
	for _, r := range f.rows {
		if _, asked := want[r.Address]; !asked {
			continue
		}
		out = append(out, db.ListEdgeFanoutMeasurementsOverRow{
			Address: r.Address, Outcome: r.Outcome, Fingerprint: r.Fingerprint,
		})
	}
	return out, nil
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

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
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

func TestReadEdgeFanoutSkipsTheMaterialReadWhereNoRowNamesACertificate(t *testing.T) {
	f := &fakeEdgeFanoutStore{
		scan:      db.Scan{Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			{Address: "198.51.100.7", Outcome: string(edgefanout.Unreachable)},
		},
	}

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
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

func TestReadEdgeFanoutReturnsTheMaterialReadsFailure(t *testing.T) {
	// A failed read is not a finding, or one unreadable signal would move every measured address.
	boom := errors.New("connection reset")
	f := &fakeEdgeFanoutStore{
		scan:      db.Scan{Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			{Address: "104.16.132.229", Outcome: string(edgefanout.Presented), Fingerprint: pgtype.Text{String: "sha256:aa", Valid: true}},
		},
		materialErr: boom,
	}

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the material read's own failure", err)
	}
	if got.Enabled || len(got.Shared) != 0 {
		t.Fatalf("got = %+v, want the zero record — a failed read decides nothing", got)
	}
}

func TestReadEdgeFanoutReadsNoMaterialWhereTheScanIsNotInForce(t *testing.T) {
	f := &fakeEdgeFanoutStore{scan: db.Scan{Kind: scan.EdgeFanoutKind, Enabled: false}}

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
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

// scopeEdge is a declaration-limb row and never a candidate: it is what a bound read leaves behind.

const (
	extensionEdge       = "104.16.132.229"
	secondExtensionEdge = "104.16.132.230"
	scopeEdge           = "23.20.0.20"
)

func boundFixtureStore(t *testing.T, der []byte) *fakeEdgeFanoutStore {
	t.Helper()
	fp := co.Fingerprint(der)
	named := func(addr string) db.ListEdgeFanoutMeasurementsRow {
		return db.ListEdgeFanoutMeasurementsRow{
			Address:     addr,
			Outcome:     string(edgefanout.Presented),
			Fingerprint: pgtype.Text{String: fp, Valid: true},
		}
	}
	return &fakeEdgeFanoutStore{
		scan:      db.Scan{Kind: scan.EdgeFanoutKind, Enabled: true},
		completed: true,
		rows: []db.ListEdgeFanoutMeasurementsRow{
			named(extensionEdge), named(secondExtensionEdge), named(scopeEdge),
		},
		material: []db.ListCertificateMaterialDERRow{{Fingerprint: fp, Der: der}},
	}
}

func TestABoundReadAsksForTheNamedAddressesAlone(t *testing.T) {
	// /scope asks about a handful of cited targets, not every address of every scope (#1036).
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver([]netip.Addr{
		netip.MustParseAddr(extensionEdge),
		netip.MustParseAddr(secondExtensionEdge),
	}))
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if f.unboundReads != 0 {
		t.Fatalf("the unbound query ran %d times under a bound read", f.unboundReads)
	}
	if len(f.boundsAsked) != 1 {
		t.Fatalf("the bound query ran %d times, want 1", len(f.boundsAsked))
	}
	want := []string{extensionEdge, secondExtensionEdge}
	if !slices.Equal(f.boundsAsked[0], want) {
		t.Fatalf("the bound query asked for %v, want %v", f.boundsAsked[0], want)
	}
	if _, measured := got.Shared[netip.MustParseAddr(scopeEdge)]; measured {
		t.Fatalf("the declaration-limb row %s reached a read bound to the extension limb", scopeEdge)
	}
	if !got.Partial {
		t.Error("a bound read returned Partial = false: the address-scope census would count over it")
	}
}

func TestAnUnboundReadIsNotPartial(t *testing.T) {
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if got.Partial {
		t.Error("an unbound read returned Partial = true: the address-scope census would refuse the whole store")
	}
}

func TestABoundReadKeepsEveryNamedAddressMeasured(t *testing.T) {
	// A dropped candidate withholds its probe in silence, so a bound may not lose a key (#1036).
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))
	candidates := []netip.Addr{
		netip.MustParseAddr(extensionEdge),
		netip.MustParseAddr(secondExtensionEdge),
	}

	whole, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
	if err != nil {
		t.Fatalf("the unbound read: %v", err)
	}
	bound, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver(candidates))
	if err != nil {
		t.Fatalf("the bound read: %v", err)
	}

	for _, addr := range candidates {
		wantShared, wantMeasured := whole.Shared[addr]
		gotShared, gotMeasured := bound.Shared[addr]
		if !wantMeasured {
			t.Fatalf("the fixture left %s unmeasured — this test would prove nothing", addr)
		}
		if !gotMeasured {
			t.Fatalf("the bound read turned the measured address %s into a pending one", addr)
		}
		if gotShared != wantShared {
			t.Fatalf("%s read as shared=%v under the bound and shared=%v unbound", addr, gotShared, wantShared)
		}
	}
}

func TestABoundReadDerivesTheSameExtensionVerdictAsAnUnboundOne(t *testing.T) {
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))
	// The errored floor asks whether any extension candidate was measured, not a declared one (#1018).
	estate := custody.Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("23.20.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions: []custody.Resolution{
			{Owner: "www.example.com", Address: netip.MustParseAddr(extensionEdge)},
			{Owner: "cdn.example.com", Address: netip.MustParseAddr(secondExtensionEdge)},
			{Owner: "edge.provider.net", Address: netip.MustParseAddr(scopeEdge)},
		},
	}
	candidates := estate.ExtensionCandidates()
	if len(candidates) != 2 {
		t.Fatalf("the fixture holds %d extension candidates, want the two in-zone edges", len(candidates))
	}

	whole, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
	if err != nil {
		t.Fatalf("the unbound read: %v", err)
	}
	bound, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver(candidates))
	if err != nil {
		t.Fatalf("the bound read: %v", err)
	}

	wide, narrow := estate.WithEdgeFanout(whole), estate.WithEdgeFanout(bound)
	for _, addr := range candidates {
		if narrow.Derive(addr) != wide.Derive(addr) {
			t.Fatalf("Derive(%s) = %s bound, %s unbound", addr, narrow.Derive(addr), wide.Derive(addr))
		}
		if narrow.MayProbe(addr, custody.ClassInternet) != wide.MayProbe(addr, custody.ClassInternet) {
			t.Fatalf("MayProbe(%s) moved under the bound: the errored floor read differently", addr)
		}
	}
	if got, want := len(narrow.ExtensionCensus()), len(wide.ExtensionCensus()); got != want {
		t.Fatalf("the census named %d entries under the bound and %d unbound", got, want)
	}
}

func TestABoundOverNoAddressIssuesNoQueryAtAll(t *testing.T) {
	// A bare nil slice would have fallen back to the unbound read, answering a question nobody asked.
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver(nil))
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if f.unboundReads != 0 || len(f.boundsAsked) != 0 {
		t.Fatalf("a bound over no address read the store: %d unbound, %d bound", f.unboundReads, len(f.boundsAsked))
	}
	if len(f.materialAsked) != 0 {
		t.Fatalf("a bound over no address read the certificate material: %v", f.materialAsked)
	}
	if !got.Enabled || len(got.Shared) != 0 {
		t.Fatalf("got = %+v, want the in-force record carrying no key", got)
	}
}

func TestAnUnboundReadCoversBothLimbs(t *testing.T) {
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutUnbounded())
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if len(f.boundsAsked) != 0 {
		t.Fatalf("the unbound read took the bound query: %v", f.boundsAsked)
	}
	for _, addr := range []string{extensionEdge, secondExtensionEdge, scopeEdge} {
		if _, measured := got.Shared[netip.MustParseAddr(addr)]; !measured {
			t.Errorf("the unbound read left %s unmeasured", addr)
		}
	}
}

func TestABoundRendersTheAddressTheWriterStores(t *testing.T) {
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))
	// A mapped candidate would match no row, and every candidate would come back pending (#1036).
	mapped := netip.MustParseAddr("::ffff:" + extensionEdge)

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver([]netip.Addr{mapped}))
	if err != nil {
		t.Fatalf("ReadEdgeFanout: %v", err)
	}
	if len(f.boundsAsked) != 1 || !slices.Equal(f.boundsAsked[0], []string{extensionEdge}) {
		t.Fatalf("the bound asked for %v, want the Unmap'ed %q", f.boundsAsked, extensionEdge)
	}
	if _, measured := got.Shared[mapped.Unmap()]; !measured {
		t.Fatalf("%s came back pending under a mapped bound", extensionEdge)
	}
}

func TestABoundReadReturnsItsFailure(t *testing.T) {
	boom := errors.New("connection reset")
	f := boundFixtureStore(t, certWithSANs(t, distinctSANs(custody.SharedEdgeThreshold)...))
	f.rowsErr = boom

	got, err := ReadEdgeFanout(context.Background(), f, EdgeFanoutOver([]netip.Addr{netip.MustParseAddr(extensionEdge)}))
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the bound read's own failure", err)
	}
	if got.Enabled || len(got.Shared) != 0 {
		t.Fatalf("got = %+v, want the zero record — a failed read decides nothing", got)
	}
}
