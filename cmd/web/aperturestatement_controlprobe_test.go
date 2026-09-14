package main

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/scan"
)

func controlProbeRowOf(t *testing.T, seeds []db.ListSeedsRow, cadence int64, read bool) apertureRowView {
	t.Helper()
	return statementRow(t, apertureStatement(readInputs().withCadence(cadence, read), seeds), controlProbeInput)
}

// A typed count stops agreeing with the prober the day ADR-0069's construction moves (SPEC §6.1).

func TestControlProbeRowStatesTheDeclaredLabelCount(t *testing.T) {
	if wildcarddiscrim.LabelCount != wildcarddiscrim.RandomLabelCount+wildcarddiscrim.StructuredLabelCount {
		t.Fatal("the label count is no longer the construction's own sum, so the chip states the wrong shape")
	}

	row := controlProbeRowOf(t, nameOnlySeeds(), testDNSCadenceSeconds, true)
	if !strings.Contains(row.State, strconv.Itoa(wildcarddiscrim.LabelCount)) {
		t.Errorf("state = %q, want the declared label count %d", row.State, wildcarddiscrim.LabelCount)
	}

	// A spelled count outlives the constant it counts, and no gate reads copy (SPEC §6.1).
	word := regexp.MustCompile(`(?i)\b(nine|ten|` + strconv.Itoa(wildcarddiscrim.LabelCount) + `|` + strconv.Itoa(wildcarddiscrim.RandomLabelCount) + `)\b`)
	for _, cell := range []string{row.CadenceWhy, row.StateDetail, row.RemedyWhy} {
		if m := word.FindString(cell); m != "" {
			t.Errorf("the copy types the construction's size %q, so it outlives it; got %q", m, cell)
		}
	}
}

// The population rides the dns Scan, whose interval is one of the two operator dials (#1883).

func TestControlProbeRowFollowsTheDNSScanInterval(t *testing.T) {
	for _, tc := range []struct {
		seconds int64
		want    string
	}{
		{testDNSCadenceSeconds, "daily"},
		{6 * 3600, "every 6 hours"},
	} {
		row := controlProbeRowOf(t, nameOnlySeeds(), tc.seconds, true)
		if row.Cadence != tc.want {
			t.Errorf("cadence at %d seconds = %q, want %q", tc.seconds, row.Cadence, tc.want)
		}
	}

	// Two rows naming one Scan must not disagree about its interval on the same screen.
	rows := apertureStatement(readInputs().withCadence(6*3600, true), nameOnlySeeds())
	qtype, control := statementRow(t, rows, queriedQtypeInput), statementRow(t, rows, controlProbeInput)
	if qtype.Cadence != control.Cadence {
		t.Errorf("the dns Scan reads %q on the qtype row and %q here, so the ledger disagrees with itself", qtype.Cadence, control.Cadence)
	}
	if !strings.Contains(control.CadenceWhy, "dns Scan") {
		t.Errorf("the cadence reason never names the carrying Scan; got %q", control.CadenceWhy)
	}
}

func TestControlProbeRowWithholdsACadenceItCouldNotRead(t *testing.T) {
	row := controlProbeRowOf(t, nameOnlySeeds(), 0, false)
	if row.Cadence != "not read" {
		t.Errorf("cadence = %q: a failed read names no interval, so the cell states none", row.Cadence)
	}
	// The set the probe asks ships with the release, so a failed dial read moves no state.
	if row.State == apertureNone {
		t.Error("a failed cadence read emptied the state cell, which reads a declared name scope")
	}
	assertNoBlankCell(t, controlProbeInput, row)
}

// An empty Seed list is a declared state, never a value we have yet to read (SPEC §2.3).

func TestControlProbeRowReadsAnEmptySeedListAsAnEmptyPopulation(t *testing.T) {
	row := controlProbeRowOf(t, nil, testDNSCadenceSeconds, true)
	if row.State != apertureNone {
		t.Errorf("state = %q on an empty Seed list, want the literal %q", row.State, apertureNone)
	}
	if row.StateKind != "fixed" {
		t.Errorf("state_kind = %q: an empty population is no failed read and no switch", row.StateKind)
	}
	if !strings.Contains(row.StateDetail, "No name scope is declared") {
		t.Errorf("the empty state never names its reason; got %q", row.StateDetail)
	}
	assertNoBlankCell(t, controlProbeInput, row)

	// A declared name scope bounds the population, so it is the one lever the cell reads.
	if named := controlProbeRowOf(t, nameOnlySeeds(), testDNSCadenceSeconds, true); named.State == apertureNone {
		t.Error("a declared name scope left the population empty, so the cell reads no bound at all")
	}
}

// An offer the operator can narrow is a finding the operator can silence (ADR-0030).

func TestControlProbeRowOffersNoRemedyAndSaysWhy(t *testing.T) {
	for _, seeds := range [][]db.ListSeedsRow{nil, nameOnlySeeds()} {
		row := controlProbeRowOf(t, seeds, testDNSCadenceSeconds, true)
		if row.Remedy != apertureNone || row.RemedyHref != "" {
			t.Errorf("remedy = %q -> %q, want `none` and no pointer", row.Remedy, row.RemedyHref)
		}
		if strings.TrimSpace(row.RemedyWhy) == "" {
			t.Fatal("`none` renders bare, and SPEC §2.3 requires its reason")
		}
		// A pointer at a screen holding no relevant control is #1854's silence in a new costume.
		if strings.Contains(row.RemedyWhy, "/scope") || strings.Contains(row.RemedyWhy, "/settings") {
			t.Errorf("the reason points at a screen while the cell says no act exists; got %q", row.RemedyWhy)
		}
	}
}

// No toggle narrows this input, so the chip must not borrow an on/off it does not have.

func TestControlProbeRowStateCarriesNoToggle(t *testing.T) {
	if kind := controlProbeRowOf(t, nameOnlySeeds(), testDNSCadenceSeconds, true).StateKind; kind != "fixed" {
		t.Errorf("state_kind = %q, want `fixed` — the chip takes no status dot", kind)
	}
}

// Every line reads declared configuration and no line reads a batch (SPEC §2.4, §8.8).

func TestControlProbeRowReadsNoEstateBesideItsNameScopes(t *testing.T) {
	bare := statementRow(t, apertureStatement(readInputs(), nameOnlySeeds()), controlProbeInput)
	peopled := statementRow(t, apertureStatement(
		readInputs().
			withStates([]db.SourceState{{Slug: scan.CTTailSource, Enabled: true}}).
			withClasses([]custody.VantageClass{custody.ClassInternet, custody.ClassInternal}),
		addressScopeSeeds(t),
	), controlProbeInput)

	if !reflect.DeepEqual(bare, peopled) {
		t.Errorf("the row moved with the estate, so it reads more than the name scopes it is bounded by:\n bare = %+v\n peopled = %+v", bare, peopled)
	}
	// The population is bounded by our own list, and a count of the estate is barred either way.
	if len(bare.Figures) != 0 {
		t.Errorf("figures = %+v; this row derives no denominator, so it states no figure", bare.Figures)
	}
	for _, cell := range []string{bare.CadenceWhy, bare.State, bare.StateDetail, bare.RemedyWhy} {
		if regexp.MustCompile(`\d+ of \d+`).MatchString(cell) {
			t.Errorf("a cell states a proportion, which SPEC §8.8 bars on this statement; got %q", cell)
		}
	}
}
