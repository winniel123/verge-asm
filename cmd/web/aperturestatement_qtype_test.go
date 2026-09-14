package main

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/scan"
)

func qtypeRowOf(t *testing.T, seconds int64, read bool) apertureRowView {
	t.Helper()
	return statementRow(t, apertureStatement(readInputs().withCadence(seconds, read), nameOnlySeeds()), queriedQtypeInput)
}

// A typed set stops agreeing with the wire the day the leaf offers an eighth qtype (SPEC §6.1).

func TestQtypeRowStatesTheLeafsOwnSet(t *testing.T) {
	offered := resolutionwalk.DefaultOffers().Qtypes
	if len(offered) == 0 {
		t.Fatal("the leaf offers no qtype, so the row has nothing to state")
	}
	names := make([]string, 0, len(offered))
	for _, q := range offered {
		names = append(names, string(q))
	}
	want := strings.Join(names, " · ")

	row := qtypeRowOf(t, testDNSCadenceSeconds, true)
	if row.State != want {
		t.Errorf("state = %q, want the leaf's set %q", row.State, want)
	}
	// A spelled size goes stale the day the leaf offers an eighth qtype, and no gate reads copy.
	for _, cell := range []string{row.StateDetail, row.RemedyWhy} {
		for _, size := range []string{"seven", strconv.Itoa(len(offered))} {
			if strings.Contains(strings.ToLower(cell), size) {
				t.Errorf("the copy types the set's size %q, so it outlives the set; got %q", size, cell)
			}
		}
	}
}

// The dns Scan is one of the two Scans carrying a cadence dial, so the cell follows it (#1883).

func TestQtypeRowCadenceFollowsTheDNSDial(t *testing.T) {
	for _, tc := range []struct {
		seconds int64
		want    string
	}{
		{testDNSCadenceSeconds, "daily"},
		{7 * 86400, "every 7 days"},
	} {
		if got := qtypeRowOf(t, tc.seconds, true).Cadence; got != tc.want {
			t.Errorf("cadence at %d seconds = %q, want %q", tc.seconds, got, tc.want)
		}
	}

	why := qtypeRowOf(t, testDNSCadenceSeconds, true).CadenceWhy
	if !strings.Contains(why, "dial") {
		t.Errorf("the cadence reason never says a dial exists; got %q", why)
	}
	// This row's exchange carries a dial, so the sibling rows' phrase would be false here.
	if strings.Contains(strings.ToLower(why), "release-coupled") {
		t.Errorf("the cadence reason calls a dialled Scan release-coupled; got %q", why)
	}
}

// A read that did not land names no cadence, and SPEC §2.3 bars the blank cell it would leave.

func TestQtypeRowNamesNoCadenceWhenTheDialDoesNotRead(t *testing.T) {
	row := qtypeRowOf(t, 0, false)
	if row.Cadence == "daily" || strings.TrimSpace(row.Cadence) == "" {
		t.Errorf("cadence = %q, want a cell naming the failed read", row.Cadence)
	}
	if row.Cadence == apertureNone {
		t.Error("`none` claims no cadence exists, and the dns Scan has one")
	}
	for _, cell := range []string{row.Input, row.Cadence, row.CadenceWhy, row.State, row.StateDetail, row.Remedy, row.RemedyWhy} {
		if strings.TrimSpace(cell) == "" {
			t.Error("a cell renders blank, and SPEC §2.3 bars that")
		}
	}
	// The set is compiled into the release, so a failed cadence read never withholds it.
	if !strings.Contains(row.State, "TXT") {
		t.Errorf("state = %q, so a failed cadence read withheld the set as well", row.State)
	}
}

// An offer the operator can narrow is a finding the operator can silence (ADR-0030).

func TestQtypeRowOffersNoRemedyAndSaysWhy(t *testing.T) {
	row := qtypeRowOf(t, testDNSCadenceSeconds, true)
	if row.Remedy != apertureNone || row.RemedyHref != "" {
		t.Errorf("remedy = %q -> %q, want `none` and no pointer", row.Remedy, row.RemedyHref)
	}
	if strings.TrimSpace(row.RemedyWhy) == "" {
		t.Fatal("`none` renders bare, and SPEC §2.3 requires its reason")
	}
	// A dial that moves the cadence moves no qtype, so it is no remedy for this input.
	if strings.Contains(row.RemedyWhy, "/scope") || strings.Contains(row.RemedyWhy, "/settings") {
		t.Errorf("the reason points at a screen while the cell says no act exists; got %q", row.RemedyWhy)
	}
}

// No toggle narrows this input, so the chip must not borrow an on/off it does not have.

func TestQtypeRowStateCarriesNoToggle(t *testing.T) {
	if kind := qtypeRowOf(t, testDNSCadenceSeconds, true).StateKind; kind != "fixed" {
		t.Errorf("state_kind = %q, want `fixed` — the chip takes no status dot", kind)
	}
}

// Every line reads declared configuration and no line reads a batch (SPEC §2.4, §8.8).

func TestQtypeRowIsConstantAcrossEstates(t *testing.T) {
	bare := statementRow(t, apertureStatement(readInputs(), nil), queriedQtypeInput)
	peopled := statementRow(t, apertureStatement(
		readInputs().
			withStates([]db.SourceState{{Slug: scan.CTTailSource, Enabled: true}}).
			withClasses([]custody.VantageClass{custody.ClassInternet, custody.ClassInternal}),
		addressScopeSeeds(t),
	), queriedQtypeInput)

	if !reflect.DeepEqual(bare, peopled) {
		t.Errorf("the row moved with the estate, so it reads more than declared configuration:\n bare = %+v\n peopled = %+v", bare, peopled)
	}
	// The row states a set rather than a count, and SPEC §2.2 makes the figure block optional.
	if len(bare.Figures) != 0 {
		t.Errorf("figures = %+v; a figure here owes a derived denominator, and none is specified", bare.Figures)
	}
}
