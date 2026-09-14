package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

func portTierFigures(t *testing.T, seeds []db.ListSeedsRow) []apertureFigureView {
	t.Helper()
	rows := apertureStatement(seeds)
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
	nameOnly := apertureStatement(nameOnlySeeds())[0]
	if nameOnly.Remedy != "Declare an address scope" || nameOnly.RemedyHref != "/scope" {
		t.Errorf("name-only remedy: got %q -> %q", nameOnly.Remedy, nameOnly.RemedyHref)
	}

	healthy := apertureStatement(addressScopeSeeds(t))[0]
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
