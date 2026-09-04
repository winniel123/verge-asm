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

func declaredScopeFixture(t *testing.T, f *fakeStore, c *http.Client, base, cidr string) {
	t.Helper()
	declare(t, c, base, "address", cidr).Body.Close()
	f.scans = append(f.scans, db.Scan{ID: 99, Kind: scan.EdgeFanoutKind, Enabled: true, CadenceSeconds: 86400})
}

func evidRow(page string) string {
	const open = `<div class="evid">`

	// An address and a CIDR both hold digits, so a whole-page search cannot tell a
	// product-chosen number from one the operator declared.
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

func TestAddressScopeCensusRowAbsentWhereTheScopeIsUnmeasured(t *testing.T) {
	// Hold-then-open carried across from the extension limb would put a pending row on
	// every address on the first day, which is noise rather than a census (ADR-0129 #956).
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

func TestAddressScopeCensusRowFiresNoMessage(t *testing.T) {
	// A Declared act stands behind the address, so the coverage-class message's safety
	// justification does not transfer to this row.
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
