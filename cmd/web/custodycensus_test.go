package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
)

// sharedEdgeDER builds one self-signed leaf whose SANs sit on SharedEdgeThreshold
// distinct registrable domains, so custody.SharedEdge reads it as a shared edge. The
// SANs reduce under `.invalid`, which RFC 2606 reserves and delegates to nobody, so
// none of them reaches real estate. The census renders the verdict, never the count,
// so the figure appears here and nowhere in the page.
func sharedEdgeDER(t *testing.T) []byte {
	t.Helper()
	return edgeDER(t, custody.SharedEdgeThreshold)
}

// edgeDER builds one self-signed leaf presenting identities for fanOut distinct
// registrable domains. It is sharedEdgeDER's parameter, so a test can pose an edge on
// either side of the boundary without a second certificate generator.
func edgeDER(t *testing.T, fanOut int) []byte {
	t.Helper()
	sans := make([]string, 0, fanOut)
	for i := 0; i < fanOut; i++ {
		sans = append(sans, fmt.Sprintf("www.d%d.invalid", i))
	}
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

// censusEstateFixture poses a custody-extended zone whose two in-zone names front two
// separate edges, and puts the `edge-fanout` Scan in force. The measurements are the
// caller's.
func censusEstateFixture(t *testing.T, f *fakeStore) {
	t.Helper()
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
	f.cited = []db.NameCitedAddressesRow{
		{SubjectKey: "shop.example.com", Address: "93.184.216.10"},
		{SubjectKey: "api.example.com", Address: "93.184.216.20"},
	}
}

// declareExtendedZone declares a name scope and turns its custody extension on, which
// is what puts its names inside the extension's reach.
func declareExtendedZone(t *testing.T, f *fakeStore, c *http.Client, base, domain string) {
	t.Helper()
	declare(t, c, base, "name", domain).Body.Close()
	for i := range f.seeds {
		if f.seeds[i].NameDomain.String == domain {
			f.seeds[i].CustodyExtension = true
			return
		}
	}
	t.Fatalf("name scope %q not declared", domain)
}

// The dual-limb row stops naming a scope the operator has EXCLUDED (ADR-0133 §1,
// #1022). The row's Scope is the declared address scope that ALSO covers the declined
// edge, and an exclusion cuts that limb — so the row becomes a bare decline.
//
// This test walks the assembler seam. custodyExtensionEstate must read the exclusions
// and carry them in through WithAddressExclusions, or this screen keeps naming a scope
// that covers the address no longer, and contradicts the address-scope census beside
// it on the same page.
func TestCustodyCensusDualLimbRowDropsAnExcludedScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	declare(t, ac, base, "address", "93.184.216.0/24").Body.Close()
	censusEstateFixture(t, f)
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
	// One measured SHARED edge, cited by an in-zone name and inside the declared
	// scope: the dual-limb row.
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "93.184.216.10", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}

	entry := func() custody.ExtensionCensusEntry {
		t.Helper()
		estate, err := custodyExtensionEstate(t.Context(), f, time.Now().UTC())
		if err != nil {
			t.Fatalf("assemble the census estate: %v", err)
		}
		for _, e := range estate.ExtensionCensus() {
			if e.Address.String() == "93.184.216.10" {
				return e
			}
		}
		t.Fatal("the declined edge holds no census row")
		return custody.ExtensionCensusEntry{}
	}

	before := entry()
	if !before.Scope.IsValid() {
		t.Fatal("the declined edge names no covering scope before the exclusion: the row below pins nothing")
	}

	excl := netip.MustParsePrefix("93.184.216.8/29")
	if _, err := f.CreateAddressExclusion(t.Context(), db.CreateAddressExclusionParams{AddressCidr: &excl, CreatedBy: 1}); err != nil {
		t.Fatalf("declare the exclusion: %v", err)
	}

	after := entry()
	if after.Scope.IsValid() {
		t.Errorf("the row still names scope %s after the operator excluded the address: the assembler dropped the exclusion read", after.Scope)
	}
	if after.State != before.State {
		t.Errorf("the decline state moved from %q to %q: an exclusion cuts the Seed limb alone and must not touch the extension's verdict", before.State, after.State)
	}
}

// A measured shared edge renders a DECLINED row naming the citing name and the
// address-scope remedy, and an unmeasured candidate renders a PENDING row. Without
// this section a veto withholds a probe with nothing on screen to say so (ADR-0129
// §5, #987).
func TestCustodyCensusDeclinedAndPendingRows(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	// .10 is measured shared. .20 carries no row at all, which is measurement
	// pending — an absence is never a value.
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "93.184.216.10", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "Edges not reached") {
		t.Fatalf("custody-extension census section absent; body: %s", page)
	}
	// The declined row: the citing name, the edge, and the remedy.
	if !strings.Contains(page, "shop.example.com") || !strings.Contains(page, "93.184.216.10") {
		t.Errorf("declined row does not name the citing name and its edge; body: %s", page)
	}
	if !strings.Contains(page, "measured as a shared edge") {
		t.Errorf("declined row does not say why the extension declined; body: %s", page)
	}
	if !strings.Contains(page, "Declare the origin addresses as an address scope, to monitor the true origin.") {
		t.Errorf("declined row states no address-scope remedy; body: %s", page)
	}
	// The pending row: the Scan is in force and has not measured this candidate, so
	// the extension holds the reach.
	if !strings.Contains(page, "api.example.com") || !strings.Contains(page, "93.184.216.20") {
		t.Errorf("pending row does not name the citing name and its edge; body: %s", page)
	}
	if !strings.Contains(page, "The extension holds the reach until it does.") {
		t.Errorf("pending row does not read as held; body: %s", page)
	}
	if !strings.Contains(page, `<span class="sc-declined">declined</span>`) ||
		!strings.Contains(page, `<span class="sc-stale">pending</span>`) {
		t.Errorf("the two row states do not carry their own chips; body: %s", page)
	}
}

// No row carries a count or a threshold, and the type is what holds that. The fan-out
// figure and the boundary it is compared against are versioned parameters of the
// `Custody` derivation, locked by the `custody/v3` corpus, so a row that rendered
// either would put a product-chosen number in front of the operator and pin the
// renderer to a parameter that is free to move (ADR-0129 §5, #987).
//
// The test reads the SHAPE rather than the rendering: an address and a CIDR both
// contain digits, so no substring search over the page can tell a product-chosen
// number from an address the operator declared.
func TestCustodyCensusRowCarriesNoNumber(t *testing.T) {
	rt := reflect.TypeOf(custodyCensusRow{})
	for i := range rt.NumField() {
		switch k := rt.Field(i).Type.Kind(); k {
		case reflect.String, reflect.Bool:
		default:
			t.Errorf("custodyCensusRow.%s is a %s — a census row carries facts, never a number",
				rt.Field(i).Name, k)
		}
	}
}

// The dual-limb row (#956): an address the extension declined and an address-scope
// `Seed` covers at once. The row states both limbs. A bare *declined* is true about the
// extension and reads as a contradiction to the person the census exists for.
func TestCustodyCensusDualLimbRow(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	declare(t, ac, base, "address", "93.184.216.0/24").Body.Close()
	censusEstateFixture(t, f)
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "93.184.216.10", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "measured as a shared edge, and covered by address scope") ||
		!strings.Contains(page, "93.184.216.0/24") {
		t.Errorf("the dual-limb row does not state both limbs; body: %s", page)
	}
	// The remedy is the OTHER row's. Telling an operator to declare an address scope
	// in the same sentence that says one already covers the address reads as a
	// contradiction, which is the very thing this row exists to avoid. What they can
	// act on here is the withdrawal, so that is what the row names.
	// The section lede carries the remedy in general terms, so the assertion names the
	// ROW's own sentence — the one TestCustodyCensusDeclinedAndPendingRows pins present
	// on a plain decline.
	if strings.Contains(page, "Declare the origin addresses as an address scope, to monitor the true origin.") {
		t.Errorf("the dual-limb row repeats a remedy already taken; body: %s", page)
	}
	if !strings.Contains(page, "Withdraw the scope and the decline takes effect.") {
		t.Errorf("the dual-limb row does not say what the decline would do; body: %s", page)
	}
}

// The section degrades honestly when the read fails. It says the measurement did not
// resolve and lists nothing — a census that fabricated a row on a database error would
// name a decline that did not happen.
func TestCustodyCensusDegradesOnReadFailure(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "93.184.216.10", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}
	f.citedErr = errors.New("resolution read failed")

	resp := get(t, ac, base+"/scope")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("scope status = %d on a failed census read, want 200 — the section degrades, the screen does not", resp.StatusCode)
	}
	page := body(t, resp)
	if !strings.Contains(page, "The fan-out measurement did not resolve on this load.") {
		t.Errorf("no honest degrade note; body: %s", page)
	}
	if strings.Contains(page, "shop.example.com") {
		t.Errorf("a row rendered from a failed read; body: %s", page)
	}
}

// A Scan that COMPLETED A BATCH and measured no extension candidate is errored on that
// limb, and the section renders the empty state (#1018). Every candidate reaches, so a
// *pending* row here would name a hold that is not happening.
//
// The store is NOT empty: it holds a declaration-limb row, which is the whole point.
// This test walks the assembler seam — custodyExtensionEstate must carry the
// measurement in through WithEdgeFanout, or the floor is never resolved and both
// candidates render as pending forever.
func TestCustodyCensusEmptyWhereTheExtensionLimbErrored(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
	// One row, on an address no in-zone name cites — the declaration limb's. Neither
	// extension candidate was measured.
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "23.20.0.20", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "No in-zone name fronts an edge the extension declined.") {
		t.Errorf("no empty state on an errored extension limb; body: %s", page)
	}
	if strings.Contains(page, `<span class="sc-stale">pending</span>`) {
		t.Errorf("a pending row rendered on an errored extension limb; body: %s", page)
	}
}

// A Scan out of force yields no row, and the section says so plainly. Nothing is
// declined and nothing is held where the measurement does not narrow — that is
// EdgeFanout's fourth absence case, and a row there would name a decline that did not
// happen.
func TestCustodyCensusEmptyWhereNothingIsDeclined(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	f.cited = []db.NameCitedAddressesRow{{SubjectKey: "shop.example.com", Address: "93.184.216.10"}}

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "No in-zone name fronts an edge the extension declined.") {
		t.Errorf("no empty state on a Scan out of force; body: %s", page)
	}
	if strings.Contains(page, `<span class="sc-declined">declined</span>`) {
		t.Errorf("a declined row rendered with the Scan out of force; body: %s", page)
	}
}

// A decline fires NO message. The register is display: a veto withholds a probe, which
// is the safe direction, and the coverage-class message exists for the dangerous one —
// the probing gate opening with no Declared act behind it. That justification does not
// transfer, so the store holds no message after a decline renders.
func TestCustodyCensusFiresNoMessage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	f.edgeFanout = []db.ListEdgeFanoutMeasurementsRow{
		{Address: "93.184.216.10", Outcome: string(edgefanout.Presented), Der: sharedEdgeDER(t)},
	}

	before := len(f.messages)
	if page := seedsBody(t, ac, base); !strings.Contains(page, "measured as a shared edge") {
		t.Fatalf("the decline did not render, so the message assertion proves nothing; body: %s", page)
	}
	if len(f.messages) != before {
		t.Errorf("messages = %d after a decline, want %d — a decline is display, never a message", len(f.messages), before)
	}
}
