package main

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
)

// The surface is the Coverage screen (coverageBody, sources_test.go), which holds the
// address scope's own membership census — the aperture meter, the one surface with a
// `Coverage` denominator (ADR-0129's #956 amendment, #989).

func declaredScopeFixture(t *testing.T, f *fakeStore, c *http.Client, base, cidr string) {
	t.Helper()
	declare(t, c, base, "address", cidr).Body.Close()
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
}

// evidRow returns the contradiction row's own markup, or the empty string where the
// meter renders none. The forbidden-content assertions read THIS rather than the page:
// an address and a CIDR both contain digits, so a search over the whole rendering
// could not tell a product-chosen number from one the operator declared.
func evidRow(page string) string {
	const open = `<div class="evid">`
	i := strings.Index(page, open)
	if i < 0 {
		return ""
	}
	rest := page[i:]
	j := strings.Index(rest, "</div>")
	if j < 0 {
		return rest
	}
	return rest[:j]
}

// A declared scope holding a measured shared edge carries the row, and the row states
// the covered count, the count above the boundary, and the exclusion remedy. Without
// it the estate probes a provider's edge while holding a measurement that says so, and
// evidence held and not shown is the fails-silently shape the model refuses.
func TestAddressScopeCensusRowStatesTheCountsAndTheRemedy(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "This scope covers 256 addresses. 1 of them presents a fan-out above the threshold.") {
		t.Errorf("the row does not state the covered count and the count above the boundary; body: %s", page)
	}
	if !strings.Contains(page, `<a href="/scope#exclusions">Exclude it from this scope</a> if it is not yours.`) {
		t.Errorf("the row does not name the exclusion remedy; body: %s", page)
	}
}

// The row reads as a sentence at either count. One shared edge is the modal case and
// the one a fixed plural would render wrong, so both are pinned.
func TestAddressScopeCensusRowReadsAsASentenceInThePlural(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	der := sharedEdgeDER(t)
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), der)
	f.measuredEdge("93.184.216.11", string(edgefanout.Presented), der)

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "This scope covers 256 addresses. 2 of them present a fan-out above the threshold.") {
		t.Errorf("the plural row does not state the two counts; body: %s", page)
	}
	if !strings.Contains(page, `<a href="/scope#exclusions">Exclude them from this scope</a> if they are not yours.`) {
		t.Errorf("the plural row does not name the exclusion remedy; body: %s", page)
	}
}

// A scope with no address above the threshold renders no row. A measured NOT-shared
// edge is measured, and it is not evidence against the declaration.
func TestAddressScopeCensusRowAbsentBelowTheThreshold(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), edgeDER(t, custody.SharedEdgeThreshold-1))

	if row := evidRow(coverageBody(t, ac, base)); row != "" {
		t.Errorf("a row rendered below the boundary: %s", row)
	}
}

// Open-then-label: an UNMEASURED declared address is probed normally and carries no
// row. Hold-then-open carried across from the extension limb by analogy would put a
// pending row on every address of every scope on the first day, which is noise rather
// than a census.
func TestAddressScopeCensusRowAbsentWhereTheScopeIsUnmeasured(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")

	page := coverageBody(t, ac, base)
	if !strings.Contains(page, "93.184.216.0/24") {
		t.Fatalf("the scope's meter did not render, so the absence proves nothing; body: %s", page)
	}
	if row := evidRow(page); row != "" {
		t.Errorf("a row rendered for an unmeasured scope: %s", row)
	}
}

// The row carries no threshold value and no verdict. The boundary stays inside the
// versioned `Custody` derivation, and *you may have over-asserted* is the sentence the
// nag test forbids. The row states two counts the operator's own estate produced, and
// nothing else.
func TestAddressScopeCensusRowCarriesNoThresholdOrVerdict(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	row := evidRow(coverageBody(t, ac, base))
	if row == "" {
		t.Fatal("the row did not render, so the content assertions prove nothing")
	}
	if strings.Contains(row, strconv.Itoa(custody.SharedEdgeThreshold)) {
		t.Errorf("the row renders the threshold: %s", row)
	}
	for _, forbidden := range []string{"over-assert", "you may have", "should not", "mistake", "wrong"} {
		if strings.Contains(strings.ToLower(row), forbidden) {
			t.Errorf("the row carries a verdict (%q): %s", forbidden, row)
		}
	}
}

// The aperture statement is unchanged. ADR-0095's meter counts what the instrument
// cannot report, and a declared shared edge is looked at and reported fine — so the
// label, the counted/total figure and the meter's own detail all survive the row.
func TestAddressScopeCensusRowLeavesTheApertureStatementAlone(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	page := coverageBody(t, ac, base)
	for _, want := range []string{
		"What the last batch walked",
		"93.184.216.0/24",
		"0 / 256 subjects",
		"address scope — the subjects still current over the enumerable addresses of the declared range",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the aperture statement lost %q; body: %s", want, page)
		}
	}
}

// The custody-extension panel is unchanged. Its subject is which addresses the
// EXTENSION pulls in, and an address-scope address is not an extension member — the
// #944 amendment kept the two registers on separate surfaces for this reason.
func TestAddressScopeCensusRowLeavesTheExtensionPanelAlone(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))
	if row := evidRow(coverageBody(t, ac, base)); row == "" {
		t.Fatal("the row did not render, so the panel assertion proves nothing")
	}

	page := seedsBody(t, ac, base)
	if strings.Contains(page, `<span class="sc-declined">declined</span>`) ||
		strings.Contains(page, `<span class="sc-stale">pending</span>`) {
		t.Errorf("a declared address-scope address reached the custody-extension panel; body: %s", page)
	}
}

// The row fires NO message. A Declared act stands behind the address, so the
// coverage-class message's safety justification does not transfer, and the nag test
// bites hardest on the operator who declared a CDN range deliberately.
func TestAddressScopeCensusRowFiresNoMessage(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	base := start(t, f, "")
	ac := login(t, base, "admin", "hunter2hunter2")
	declaredScopeFixture(t, f, ac, base, "93.184.216.0/24")
	f.measuredEdge("93.184.216.10", string(edgefanout.Presented), sharedEdgeDER(t))

	before := len(f.messages)
	if row := evidRow(coverageBody(t, ac, base)); row == "" {
		t.Fatal("the row did not render, so the message assertion proves nothing")
	}
	if len(f.messages) != before {
		t.Errorf("messages = %d after the row rendered, want %d — the row is display, never a message", len(f.messages), before)
	}
}
