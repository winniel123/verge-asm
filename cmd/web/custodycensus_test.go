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

func sharedEdgeDER(t *testing.T) []byte {
	t.Helper()
	return edgeDER(t, custody.SharedEdgeThreshold)
}

func edgeDER(t *testing.T, fanOut int) []byte {
	t.Helper()
	sans := make([]string, 0, fanOut)
	for i := 0; i < fanOut; i++ {
		// RFC 2606 reserves `.invalid` and delegates it to nobody, so no SAN reaches real estate.
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

func censusEstateFixture(t *testing.T, f *fakeStore) {
	t.Helper()
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
	f.cited = []db.NameCitedAddressesRow{
		{SubjectKey: "shop.example.com", Address: "93.184.216.10"},
		{SubjectKey: "api.example.com", Address: "93.184.216.20"},
	}
}

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

func TestCustodyCensusDualLimbRowDropsAnExcludedScope(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	declare(t, ac, base, "address", "93.184.216.0/24").Body.Close()
	censusEstateFixture(t, f)
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
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

func TestCustodyCensusDeclinedRowAndHeldLine(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "Edges not reached") {
		t.Fatalf("custody-extension census section absent; body: %s", page)
	}
	if !strings.Contains(page, "shop.example.com") || !strings.Contains(page, "93.184.216.10") {
		t.Errorf("declined row does not name the citing name and its edge; body: %s", page)
	}
	if !strings.Contains(page, "measured as a shared edge") {
		t.Errorf("declined row does not say why the extension declined; body: %s", page)
	}
	if !strings.Contains(page, "Declare the origin addresses as an address scope, to monitor the true origin.") {
		t.Errorf("declined row states no address-scope remedy; body: %s", page)
	}
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
	if !strings.Contains(page, censusCountChip(1)) {
		t.Errorf("the count chip does not count the listed decline; body: %s", page)
	}
}

func censusCountChip(n int) string {
	// Other cards on this screen carry a bare `sc-count`, so an unanchored match reads theirs.
	return fmt.Sprintf(`Edges not reached</h3></div><span class="sc-count">%d</span>`, n)
}

func TestCustodyCensusHeldCandidatesCollapseToOneLine(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
	f.cited = []db.NameCitedAddressesRow{
		{SubjectKey: "a.example.com", Address: "93.184.216.10"},
		{SubjectKey: "b.example.com", Address: "93.184.216.10"},
		{SubjectKey: "c.example.com", Address: "93.184.216.20"},
	}

	page := seedsBody(t, ac, base)
	if got := strings.Count(page, `<span class="sc-stale">pending</span>`); got != 1 {
		t.Errorf("pending chips = %d, want 1 — the held candidates collapse to one line", got)
	}
	if !strings.Contains(page, `<span class="mono">2</span> edges await a first`) {
		t.Errorf("the held line does not count the distinct held edges; body: %s", page)
	}
	if strings.Contains(page, `<span class="sc-declined">declined</span>`) {
		t.Errorf("a declined chip rendered with nothing measured; body: %s", page)
	}
	for n := range 4 {
		if strings.Contains(page, censusCountChip(n)) {
			t.Errorf("the card carries a count chip reading %d with no row listed; body: %s", n, page)
		}
	}
	if strings.Contains(page, "These in-zone names front an edge the custody extension does not reach.") {
		t.Errorf("the decline lede rendered with no decline to lead; body: %s", page)
	}
}

func TestToCustodyCensusViewSplitsDeclinesFromHeld(t *testing.T) {
	entries := []custody.ExtensionCensusEntry{
		{Name: "a.example.com", Address: netip.MustParseAddr("93.184.216.10"), State: custody.ExtensionPending},
		{
			Name: "b.example.com", Address: netip.MustParseAddr("93.184.216.20"),
			State: custody.ExtensionDeclined, Scope: netip.MustParsePrefix("93.184.216.0/24"),
		},
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
	if strings.Contains(page, "Declare the origin addresses as an address scope, to monitor the true origin.") {
		t.Errorf("the dual-limb row repeats a remedy already taken; body: %s", page)
	}
	if !strings.Contains(page, "Withdraw the scope and the decline takes effect.") {
		t.Errorf("the dual-limb row does not say what the decline would do; body: %s", page)
	}
}

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

func TestCustodyCensusEmptyWhereTheExtensionLimbErrored(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declareExtendedZone(t, f, ac, base, "example.com")
	censusEstateFixture(t, f)
	f.completedBatchKinds[scan.EdgeFanoutKind] = true
	f.measuredEdge("23.20.0.20", string(edgefanout.Presented), sharedEdgeDER(t))

	page := seedsBody(t, ac, base)
	if !strings.Contains(page, "No in-zone name fronts an edge the extension declined.") {
		t.Errorf("no empty state on an errored extension limb; body: %s", page)
	}
	if strings.Contains(page, `<span class="sc-stale">pending</span>`) {
		t.Errorf("a pending row rendered on an errored extension limb; body: %s", page)
	}
}

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
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), shared)
	f.measuredEdge("93.184.216.20", string(edgefanout.Presented), shared)
	// No in-zone name cites this third address, so a bound widened back to the store shows it.
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

	if entries := estate.AddressScopeCensus(); entries != nil {
		t.Errorf("the address-scope census counted %+v over the Scope render's bound estate: "+
			"a short count states a number this install did not measure", entries)
	}
}
