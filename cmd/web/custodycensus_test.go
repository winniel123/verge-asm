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
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

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
// address-scope remedy, and an unmeasured candidate is counted on the held line.
// Without this section a veto withholds a probe with nothing on screen to say so
// (ADR-0129 §5, #987).
func TestCustodyCensusDeclinedRowAndHeldLine(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	// .10 is measured shared. .20 carries no row at all, which is measurement
	// pending — an absence is never a value.
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

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
	// The held candidate: the Scan is in force and has not measured it, so the
	// extension holds the reach. It is stated once, and it names nobody — see
	// TestCustodyCensusHeldCandidatesCollapseToOneLine for why.
	if !strings.Contains(page, "One edge awaits a first") {
		t.Errorf("the held candidate is not stated; body: %s", page)
	}
	if !strings.Contains(page, "The extension holds the reach until the scan runs") {
		t.Errorf("the held line does not read as held; body: %s", page)
	}
	if !strings.Contains(page, `<span class="sc-declined">declined</span>`) ||
		!strings.Contains(page, `<span class="sc-stale">pending</span>`) {
		t.Errorf("the two states do not carry their own chips; body: %s", page)
	}
	// The count chip counts the ROWS listed, so it counts the one decline. The held
	// line carries its own number and is one line: a chip that summed the two would
	// add citing names to edges, which this section counts differently on purpose.
	if !strings.Contains(page, censusCountChip(1)) {
		t.Errorf("the count chip does not count the listed decline; body: %s", page)
	}
}

// censusCountChip is the custody-extension card's count chip, anchored to its own
// heading. Bare `sc-count` appears on other cards of the same screen, so an assertion
// that did not anchor would read another section's number.
func censusCountChip(n int) string {
	return fmt.Sprintf(`Edges not reached</h3></div><span class="sc-count">%d</span>`, n)
}

// The held candidates COLLAPSE to one line, and the declines keep a row each (#1015).
//
// A pending candidate carries no remedy: the operator cannot act on a measurement that
// has not happened, and the state clears within one Scan cadence with no act of theirs.
// A row each would make this section's WORST render its FIRST — the Scan ships enabled
// and measures nothing until its first Batch completes, so a zone holding thousands of
// in-estate names would render thousands of rows on the first load of /scope, all of
// them the same fact.
//
// The declines are the OTHER case: the citing name is the one thing the operator acts
// on, so each keeps its own row however many there are.
func TestCustodyCensusHeldCandidatesCollapseToOneLine(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
	// Three in-zone names, TWO edges: the first two names front the same one. Nothing
	// is measured, so every candidate is held.
	f.cited = []db.NameCitedAddressesRow{
		{SubjectKey: "a.example.com", Address: "93.184.216.10"},
		{SubjectKey: "b.example.com", Address: "93.184.216.10"},
		{SubjectKey: "c.example.com", Address: "93.184.216.20"},
	}

	page := seedsBody(t, ac, base)
	if got := strings.Count(page, `<span class="sc-stale">pending</span>`); got != 1 {
		t.Errorf("pending chips = %d, want 1 — the held candidates collapse to one line", got)
	}
	// TWO, not three. The derivation emits one entry per (citing name, edge) pair, and
	// the line counts EDGES: two names fronting one held edge is one thing waiting.
	if !strings.Contains(page, `<span class="mono">2</span> edges await a first`) {
		t.Errorf("the held line does not count the distinct held edges; body: %s", page)
	}
	// Nothing is measured, so nothing is declined and no row is listed.
	if strings.Contains(page, `<span class="sc-declined">declined</span>`) {
		t.Errorf("a declined chip rendered with nothing measured; body: %s", page)
	}
	// The chip counts the rows listed, and nothing is listed, so the card carries none.
	for n := range 4 {
		if strings.Contains(page, censusCountChip(n)) {
			t.Errorf("the card carries a count chip reading %d with no row listed; body: %s", n, page)
		}
	}
	// The lede carries the address-scope remedy, which belongs to a decline alone.
	// Held candidates have no remedy, so the lede must not appear over them.
	if strings.Contains(page, "These in-zone names front an edge the custody extension does not reach.") {
		t.Errorf("the decline lede rendered with no decline to lead; body: %s", page)
	}
}

// toCustodyCensusView splits the derivation's entries: a decline becomes a row, and a
// held EDGE becomes one increment of the count however many names front it (#1015).
// The derivation still yields every candidate — the collapse is a display choice, made
// in the web layer alone, so internal/custody keeps naming them all.
//
// The two counts run on different units on purpose. A decline is per citing name,
// because the name is what the operator acts on. A hold is per edge, because two names
// fronting one held edge is ONE thing waiting, and a line reading two would inflate the
// only fact it carries.
func TestToCustodyCensusViewSplitsDeclinesFromHeld(t *testing.T) {
	entries := []custody.ExtensionCensusEntry{
		{Name: "a.example.com", Address: netip.MustParseAddr("93.184.216.10"), State: custody.ExtensionPending},
		{
			Name: "b.example.com", Address: netip.MustParseAddr("93.184.216.20"),
			State: custody.ExtensionDeclined, Scope: netip.MustParsePrefix("93.184.216.0/24"),
		},
		// The same held edge as the first entry, fronted by a second name.
		{Name: "c.example.com", Address: netip.MustParseAddr("93.184.216.10"), State: custody.ExtensionPending},
		{Name: "d.example.com", Address: netip.MustParseAddr("93.184.216.30"), State: custody.ExtensionPending},
	}

	view := toCustodyCensusView(entries)
	want := []custodyCensusRow{{Name: "b.example.com", Address: "93.184.216.20", Scope: "93.184.216.0/24"}}
	if !reflect.DeepEqual(view.Rows, want) {
		t.Errorf("rows = %+v, want %+v — only a decline earns a row", view.Rows, want)
	}
	if view.Pending != 2 {
		t.Errorf("pending = %d, want 2 — the count is of held EDGES, and two names front one of them", view.Pending)
	}
}

// A state this render does not know is SKIPPED, never absorbed into the held count.
// ExtensionState is a string type, so a state added later would otherwise be asserted
// on screen as *awaiting a first measurement* — a claim about the Scan's progress that
// nothing measured. The section states what it holds and stays silent about the rest.
func TestToCustodyCensusViewSkipsAnUnknownState(t *testing.T) {
	view := toCustodyCensusView([]custody.ExtensionCensusEntry{
		{Name: "a.example.com", Address: netip.MustParseAddr("93.184.216.10"), State: custody.ExtensionState("reconsidered")},
		{Name: "b.example.com", Address: netip.MustParseAddr("93.184.216.20")},
	})
	if len(view.Rows) != 0 {
		t.Errorf("rows = %+v, want none — an unknown state is not a decline", view.Rows)
	}
	if view.Pending != 0 {
		t.Errorf("pending = %d, want 0 — an unknown state is not a held edge", view.Pending)
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
//
// The held count of #1015 sits on custodyCensusView and NOT here, which is what keeps
// this assertion true. That number is a fact this install measured — how many of its
// own candidates wait — never a number the product chose, so §5 does not reach it.
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
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

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
	// ROW's own sentence — the one TestCustodyCensusDeclinedRowAndHeldLine pins present
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
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))
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
	f.measuredEdge("23.20.0.20", string(edgefanout.Presented), sharedEdgeDER(t))

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
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	before := len(f.messages)
	if page := seedsBody(t, ac, base); !strings.Contains(page, "measured as a shared edge") {
		t.Fatalf("the decline did not render, so the message assertion proves nothing; body: %s", page)
	}
	if len(f.messages) != before {
		t.Errorf("messages = %d after a decline, want %d — a decline is display, never a message", len(f.messages), before)
	}
}

// `/scope` BINDS its measurement read to its own extension candidates, and `/coverage`
// does not (#1036). This test walks the assembler seam for both, so a session that
// swapped either bound is told here.
//
// The bound is the whole point of the ticket: the unbound read pulled every measured
// address of every declared address scope on a render whose census asks about a handful
// of cited direct-A targets. `edge_fanout_observation` is never pruned (#985), so that
// read grows with the estate AND with time.
//
// The fixture holds a declaration-limb row no in-zone name cites, so a bound that had
// quietly widened back to the whole store shows up as that address arriving.
func TestTheScopeCensusBindsItsFanOutReadAndTheCoverageCensusDoesNot(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	declare(t, ac, base, "address", "23.20.0.0/24").Body.Close()
	censusEstateFixture(t, f)
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
	shared := sharedEdgeDER(t)
	// Two extension candidates, and one declaration-limb address no in-zone name cites.
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), shared)
	f.measuredEdge("93.184.216.20", string(edgefanout.Presented), shared)
	f.measuredEdge("23.20.0.30", string(edgefanout.Presented), shared)

	estate, err := custodyExtensionEstate(t.Context(), f, time.Now().UTC())
	if err != nil {
		t.Fatalf("assemble the census estate: %v", err)
	}
	if len(f.edgeFanoutBounds) != 1 {
		t.Fatalf("the Scope render issued %d bound reads, want 1 — it must not take the unbound query",
			len(f.edgeFanoutBounds))
	}
	want := estate.ExtensionCandidates()
	if len(f.edgeFanoutBounds[0]) != len(want) {
		t.Fatalf("the bound named %v, want the extension candidates %v", f.edgeFanoutBounds[0], want)
	}
	for i, addr := range want {
		if f.edgeFanoutBounds[0][i] != addr.String() {
			t.Fatalf("the bound named %v, want the extension candidates %v", f.edgeFanoutBounds[0], want)
		}
	}
	// The census still declines both candidates. A bound that had dropped one would turn
	// it from measured into pending, which is a HOLD and renders no row at all.
	declined := map[string]bool{}
	for _, e := range estate.ExtensionCensus() {
		if e.State == custody.ExtensionDeclined {
			declined[e.Address.String()] = true
		}
	}
	for _, addr := range []string{"93.184.216.10", "93.184.216.20"} {
		if !declined[addr] {
			t.Errorf("%s did not decline under the bound read: the bound turned a measured edge into a held one", addr)
		}
	}

	// `/coverage` reads the DECLARATION limb, whose candidate set already is most of the
	// store, so it takes the unbound query and lands on no bound at all.
	calls := len(f.edgeFanoutBounds)
	got, err := addressScopeSharedEdges(t.Context(), f)
	if err != nil {
		t.Fatalf("read the address-scope census: %v", err)
	}
	if len(f.edgeFanoutBounds) != calls {
		t.Errorf("the address-scope census bound its read to %v: the declaration limb takes the unbound query (#1036)",
			f.edgeFanoutBounds[calls:])
	}
	scope := netip.MustParsePrefix("23.20.0.0/24")
	if got[scope] != 1 {
		t.Errorf("the declared scope counted %d shared edges, want 1 — the unbound read lost its own limb's row", got[scope])
	}

	// The backstop, at the seam the two surfaces share. Handing the Scope render's
	// estate to the address-scope census yields NO ENTRY rather than a count short by
	// every declaration-limb row its bound left behind (#1036).
	if entries := estate.AddressScopeCensus(); entries != nil {
		t.Errorf("the address-scope census counted %+v over the Scope render's bound estate: "+
			"a short count states a number this install did not measure", entries)
	}
}
