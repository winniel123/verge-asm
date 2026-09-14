package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

const vantageClassInput = "Vantage class"

func statementRow(t *testing.T, rows []apertureRowView, input string) apertureRowView {
	t.Helper()
	for _, r := range rows {
		if r.Input == input {
			return r
		}
	}
	t.Fatalf("apertureStatement: no %q row", input)
	return apertureRowView{}
}

func vantageClassRowOf(t *testing.T, classes ...custody.VantageClass) apertureRowView {
	t.Helper()
	return statementRow(t, apertureStatement(nil, classes, true), vantageClassInput)
}

func portTierFigures(t *testing.T, seeds []db.ListSeedsRow) []apertureFigureView {
	t.Helper()
	rows := apertureStatement(seeds, nil, true)
	for _, r := range rows {
		if r.Input == "Port and transport tiers" {
			if len(r.Figures) != 3 {
				t.Fatalf("port-tier figures: want 3, got %d", len(r.Figures))
			}
			return r.Figures
		}
	}
	t.Fatal("apertureStatement: no port-tier row")
	return nil
}

func nameOnlySeeds() []db.ListSeedsRow {
	return []db.ListSeedsRow{{Kind: "name"}}
}

func addressScopeSeeds(t *testing.T) []db.ListSeedsRow {
	t.Helper()
	p := mustPrefix(t, "203.0.113.0/24")
	return []db.ListSeedsRow{{Kind: "name"}, {Kind: "address", AddressCidr: &p}}
}

func figureDenominator(t *testing.T, text string) int {
	t.Helper()
	fields := strings.Fields(text)
	if len(fields) < 3 || fields[1] != "of" {
		t.Fatalf("figure %q: want a %q form", text, "N of D …")
	}
	d, err := strconv.Atoi(fields[2])
	if err != nil {
		t.Fatalf("figure %q: denominator: %v", text, err)
	}
	return d
}

func figureNumerator(t *testing.T, text string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.Fields(text)[0])
	if err != nil {
		t.Fatalf("figure %q: numerator: %v", text, err)
	}
	return n
}

// Test 1 of SPEC docs/spec/aperture-statement.md §6.2 — the list-movement gate.

func TestApertureStatementDenominatorsAreDerived(t *testing.T) {
	wantPairs := vergecore.Default().Count().Sensitive
	wantRules := len(signal.AllRuleNames())

	for _, tc := range []struct {
		name  string
		seeds []db.ListSeedsRow
	}{
		{"name only", nameOnlySeeds()},
		{"address scope", addressScopeSeeds(t)},
	} {
		figs := portTierFigures(t, tc.seeds)
		for i, want := range []int{wantPairs, wantPairs, wantRules} {
			if got := figureDenominator(t, figs[i].Text); got != want {
				t.Errorf("%s: figure %d denominator: got %d, want %d (%q)", tc.name, i+1, got, want, figs[i].Text)
			}
		}
	}
}

// Test 2 of SPEC docs/spec/aperture-statement.md §6.2 — the floor.

func TestApertureStatementUnreadNumeratorHoldsItsFloor(t *testing.T) {
	core := vergecore.Default()
	sensitive := core.Count().Sensitive
	udp := sensitiveUDPPairs(core)
	if udp == 0 {
		t.Fatal("the sensitive half holds no UDP pair, so the floor this test guards is gone")
	}

	p6 := mustPrefix(t, "2001:db8::/32")
	for _, tc := range []struct {
		name  string
		seeds []db.ListSeedsRow
		want  int
	}{
		{"no seed at all", nil, sensitive},
		{"name only", nameOnlySeeds(), sensitive},
		{"one address scope", addressScopeSeeds(t), udp},
		{"a second address scope", append(addressScopeSeeds(t), db.ListSeedsRow{Kind: "address", AddressCidr: &p6}), udp},
		{"a name scope with the custody extension on", []db.ListSeedsRow{{Kind: "name", CustodyExtension: true}}, udp},
		{"an address row carrying no range", []db.ListSeedsRow{{Kind: "address"}}, sensitive},
	} {
		figs := portTierFigures(t, tc.seeds)
		got := figureNumerator(t, figs[0].Text)
		if got != tc.want {
			t.Errorf("%s: unread numerator: got %d, want %d (%q)", tc.name, got, tc.want, figs[0].Text)
		}
		if got != sensitive && got != udp {
			t.Errorf("%s: unread numerator %d is neither %d nor the floor %d", tc.name, got, sensitive, udp)
		}
	}
}

// ADR-0071 §4 makes this one rule dark without an internet vantage, and a declared vantage clears
// that state, so the rule can still speak and figure 3 stays at zero (#1918).

const darkWithoutAnInternetVantage = "non-globally-reachable-address-resolved-from-internet"

// Test 3 of SPEC docs/spec/aperture-statement.md §6.2 — the domain guard behind figure 3's zero.

func TestEveryRuleSendsConfigurationAbsenceOutsideTheDomain(t *testing.T) {
	check := func(name string, got signal.Outcome) {
		t.Helper()
		if name == darkWithoutAnInternetVantage {
			if got != signal.NotEvaluable {
				t.Errorf("%s: the documented exception no longer fires; drop it from this test", name)
			}
			return
		}
		if got == signal.NotEvaluable {
			t.Errorf("%s: unconfigured facts: got NotEvaluable, want OutsideDomain", name)
		}
	}

	rules := 0
	for _, r := range signal.All() {
		check(r.Name(), r.Eval(signal.NameFacts{Name: "n"}))
		rules++
	}
	for _, r := range signal.AllEndpointRules() {
		check(r.Name(), r.Eval(signal.EndpointFacts{Subject: "e"}))
		rules++
	}
	for _, r := range signal.AllServiceRules() {
		check(r.Name(), r.Eval(signal.ServiceFacts{Subject: "s"}))
		rules++
	}
	if rules != len(signal.AllRuleNames()) {
		t.Errorf("swept %d rules, but the corpus names %d", rules, len(signal.AllRuleNames()))
	}
}

func TestApertureStatementRemedySwitchesOnTheDeclaredLever(t *testing.T) {
	nameOnly := apertureStatement(nameOnlySeeds(), nil, true)[0]
	if nameOnly.Remedy != "Declare an address scope" || nameOnly.RemedyHref != "/scope" {
		t.Errorf("name-only remedy: got %q -> %q", nameOnly.Remedy, nameOnly.RemedyHref)
	}

	healthy := apertureStatement(addressScopeSeeds(t), nil, true)[0]
	if healthy.Remedy != apertureNone || healthy.RemedyHref != "" {
		t.Errorf("address-scope remedy: got %q -> %q, want %q and no link", healthy.Remedy, healthy.RemedyHref, apertureNone)
	}

	for _, row := range []apertureRowView{nameOnly, healthy} {
		for _, cell := range []string{row.Input, row.Cadence, row.CadenceWhy, row.State, row.StateDetail, row.Remedy, row.RemedyWhy} {
			if strings.TrimSpace(cell) == "" {
				t.Errorf("%q: a cell renders blank, and SPEC §2.3 bars that", row.Input)
			}
		}
	}
}

// SPEC docs/spec/aperture-statement.md §4.2 — the remedy is total over the two Exposure legs.

func TestVantageClassRemedyIsTotalOverTheTwoExposureLegs(t *testing.T) {
	net, ral, unv := custody.ClassInternet, custody.ClassInternal, custody.ClassUnverified

	for _, tc := range []struct {
		name    string
		classes []custody.VantageClass
		remedy  string
		href    string
		why     string
	}{
		{
			"both legs", []custody.VantageClass{net, ral}, apertureNone, "",
			"A vantage reads from each side of your boundary, so no class is missing.",
		},
		{
			"unverified beside both legs", []custody.VantageClass{unv, net, ral}, apertureNone, "",
			"A vantage reads from each side of your boundary, so no class is missing.",
		},
		{
			"the internet leg alone", []custody.VantageClass{net, net}, "Provision a prober inside your estate", apertureVantagesHref,
			"No declared address scope covers any prober, so no vantage starts the internal leg. Run a prober inside your estate, then declare its egress as an address scope.",
		},
		{
			"the internal leg alone", []custody.VantageClass{ral}, "Provision a prober", apertureVantagesHref,
			"No vantage presents an address outside your declared scopes, so no vantage starts the internet leg. Exposure needs an outside observer, unconditionally.",
		},
		{
			"unverified alone", []custody.VantageClass{unv}, "Provision a prober", apertureVantagesHref,
			"No vantage presents an observed address, so neither leg has a reader. Exposure needs an outside observer first.",
		},
		{
			"no vantage at all", nil, "Provision a prober", apertureVantagesHref,
			"No vantage presents an observed address, so neither leg has a reader. Exposure needs an outside observer first.",
		},
	} {
		row := vantageClassRowOf(t, tc.classes...)
		if row.Remedy != tc.remedy || row.RemedyHref != tc.href {
			t.Errorf("%s: remedy = %q -> %q, want %q -> %q", tc.name, row.Remedy, row.RemedyHref, tc.remedy, tc.href)
		}
		if row.RemedyWhy != tc.why {
			t.Errorf("%s: reason = %q, want %q", tc.name, row.RemedyWhy, tc.why)
		}
		for _, cell := range []string{row.Input, row.Cadence, row.CadenceWhy, row.State, row.StateDetail, row.Remedy, row.RemedyWhy} {
			if strings.TrimSpace(cell) == "" {
				t.Errorf("%s: a cell renders blank, and SPEC §2.3 bars that", tc.name)
			}
		}
	}
}

func TestVantageClassStateCountsEveryDerivedClass(t *testing.T) {
	net, ral, unv := custody.ClassInternet, custody.ClassInternal, custody.ClassUnverified

	for _, tc := range []struct {
		classes []custody.VantageClass
		want    string
	}{
		{[]custody.VantageClass{net, ral, ral}, "1 internet · 2 internal"},
		{[]custody.VantageClass{ral, ral}, "2 internal"},
		{[]custody.VantageClass{unv}, "1 unverified"},
		{[]custody.VantageClass{ral, unv, net}, "1 internet · 1 internal · 1 unverified"},
		// A one-prober install and a ten-prober one must not read alike (SPEC §4.1).
		{[]custody.VantageClass{net}, "1 internet"},
		// A class the derivation gains renders last, and never drops out of the count.
		{[]custody.VantageClass{"mixed", net}, "1 internet · 1 mixed"},
		{[]custody.VantageClass{net, net, net, net, net, net, net, net, net, net}, "10 internet"},
	} {
		if got := vantageClassRowOf(t, tc.classes...).State; got != tc.want {
			t.Errorf("state for %v: got %q, want %q", tc.classes, got, tc.want)
		}
	}

	empty := vantageClassRowOf(t)
	if empty.State != apertureNone {
		t.Errorf("state with no declared vantage: got %q, want %q", empty.State, apertureNone)
	}
	if strings.TrimSpace(empty.StateDetail) == "" {
		t.Error("an empty cell renders `none` plus its reason, and SPEC §2.3 bars the bare word")
	}
}

func TestVantageClassCadenceIsNoneAndNeverEveryBatch(t *testing.T) {
	row := vantageClassRowOf(t, custody.ClassInternet)
	if row.Cadence != apertureNone {
		t.Errorf("cadence = %q, want %q: a value derived at the point of use has no currency (ADR-0028)", row.Cadence, apertureNone)
	}
	if strings.Contains(strings.ToLower(row.CadenceWhy), "every batch") {
		t.Errorf("the cadence reason reads %q, which restates the withdrawn CONTEXT.md claim (#1896)", row.CadenceWhy)
	}
}

func TestVantageClassRowWithholdsWhatItCouldNotRead(t *testing.T) {
	row := statementRow(t, apertureStatement(nil, nil, false), vantageClassInput)
	if row.State == apertureNone {
		t.Error("a failed read renders as `none`, which claims no vantage is declared")
	}
	if row.Remedy != apertureNone || row.RemedyHref != "" {
		t.Errorf("remedy = %q -> %q: a failed read names no missing leg, so it names no act", row.Remedy, row.RemedyHref)
	}
	for _, cell := range []string{row.Input, row.Cadence, row.CadenceWhy, row.State, row.StateDetail, row.Remedy, row.RemedyWhy} {
		if strings.TrimSpace(cell) == "" {
			t.Error("a withheld row renders a blank cell, and SPEC §2.3 bars that")
		}
	}
}

// The ledger's order is SPEC §2.5's, and the class row is row 6 to the port tier's row 2.

func TestVantageClassRowFollowsThePortTierRow(t *testing.T) {
	rows := apertureStatement(nameOnlySeeds(), nil, true)
	order := make([]string, 0, len(rows))
	for _, r := range rows {
		order = append(order, r.Input)
	}
	want := []string{"Port and transport tiers", vantageClassInput}
	if strings.Join(order, "|") != strings.Join(want, "|") {
		t.Errorf("ledger order = %v, want %v", order, want)
	}
}
