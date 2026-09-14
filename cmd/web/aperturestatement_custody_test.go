package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

func custodyGateRowOf(t *testing.T, seeds ...db.ListSeedsRow) apertureRowView {
	t.Helper()
	return statementRow(t, apertureStatement(nil, true, seeds, testDNSCadenceSeconds, true, nil, true), custodyGateInput)
}

func nameSeed(extended bool) db.ListSeedsRow {
	return db.ListSeedsRow{Kind: "name", CustodyExtension: extended}
}

// SPEC docs/spec/aperture-statement.md §2.4 — the state reads the declared column, never a batch.

func TestCustodyGateStateReadsTheDeclaredExtension(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	addr := db.ListSeedsRow{Kind: "address", AddressCidr: &p}

	for _, tc := range []struct {
		name   string
		seeds  []db.ListSeedsRow
		state  string
		kind   string
		figure string
	}{
		{"no seed at all", nil, "total · extension off", "off", ""},
		{"an address scope alone", []db.ListSeedsRow{addr}, "total · extension off", "off", ""},
		{"one name scope, unextended", []db.ListSeedsRow{nameSeed(false)}, "total · extension off", "off", "0 of 1 name scope extended"},
		{"one name scope, extended", []db.ListSeedsRow{nameSeed(true)}, "total · extension on", "on", "1 of 1 name scope extended"},
		// A chip reading `on` over a scope still unextended asserts a binary the switch is not.
		{"one of two extended", []db.ListSeedsRow{nameSeed(true), nameSeed(false), addr}, "total · extension partial", "off", "1 of 2 name scopes extended"},
		{"both extended", []db.ListSeedsRow{nameSeed(true), nameSeed(true)}, "total · extension on", "on", "2 of 2 name scopes extended"},
	} {
		row := custodyGateRowOf(t, tc.seeds...)
		if row.State != tc.state || row.StateKind != tc.kind {
			t.Errorf("%s: state = %q (%s), want %q (%s)", tc.name, row.State, row.StateKind, tc.state, tc.kind)
		}
		switch {
		case tc.figure == "" && len(row.Figures) != 0:
			t.Errorf("%s: figures = %+v: no name scope is declared, so the row counts nothing", tc.name, row.Figures)
		case tc.figure != "" && (len(row.Figures) != 1 || row.Figures[0].Text != tc.figure):
			t.Errorf("%s: figures = %+v, want the single %q", tc.name, row.Figures, tc.figure)
		}
		assertNoBlankCell(t, tc.name, row)
	}
}

// SPEC docs/spec/aperture-statement.md §3.4 — a pointer ships only where an act genuinely exists.

func TestCustodyGateRemedyPointsAtTheExtensionControl(t *testing.T) {
	for _, tc := range []struct {
		name   string
		seeds  []db.ListSeedsRow
		remedy string
		href   string
		why    string
	}{
		{
			"no name scope to extend", nil,
			"Declare a name scope", apertureScopeHref,
			"A custody extension is a property of a name scope. Declare a name scope, then extend custody to the addresses it resolves into.",
		},
		{
			"a name scope carrying no extension", []db.ListSeedsRow{nameSeed(false)},
			"Extend custody to a name scope", apertureScopeHref,
			"No name scope carries the extension, so the gate admits an address only where a declared address scope covers it. A switch on the Scope screen extends custody to the addresses a name scope resolves into.",
		},
		{
			"one name scope still unextended", []db.ListSeedsRow{nameSeed(true), nameSeed(false)},
			"Extend custody to a name scope", apertureScopeHref,
			"A switch on the Scope screen reaches each name scope that does not carry the extension yet.",
		},
		{
			"every name scope extended", []db.ListSeedsRow{nameSeed(true)},
			apertureNone, "",
			"Every declared name scope carries the extension, so no further switch widens this gate.",
		},
	} {
		row := custodyGateRowOf(t, tc.seeds...)
		if row.Remedy != tc.remedy || row.RemedyHref != tc.href {
			t.Errorf("%s: remedy = %q -> %q, want %q -> %q", tc.name, row.Remedy, row.RemedyHref, tc.remedy, tc.href)
		}
		if row.RemedyWhy != tc.why {
			t.Errorf("%s: reason = %q, want %q", tc.name, row.RemedyWhy, tc.why)
		}
	}
}

// Cadence setters ship for `dns` and `zone` alone, so this row is release-coupled (#1883).

func TestCustodyGateCadenceNamesTheDispatchAndDeniesADial(t *testing.T) {
	row := custodyGateRowOf(t, nameSeed(true))
	if row.Cadence != "every dispatch · daily" {
		t.Errorf("cadence = %q, want the dispatch gate beside the edge-fanout Scan", row.Cadence)
	}
	for _, want := range []string{"edge-fanout", "Release-coupled", "dns and zone"} {
		if !strings.Contains(row.CadenceWhy, want) {
			t.Errorf("the cadence reason omits %q, so it never says whether a dial exists; got %q", want, row.CadenceWhy)
		}
	}
}

// The queried address scope is a lever on this gate, not a peer of it (ADR-0079, #1906).

func TestCustodyGateIsOneRowAndNamesTheAddressScopeLever(t *testing.T) {
	gates := 0
	for _, r := range apertureStatement(nil, true, addressScopeSeeds(t), testDNSCadenceSeconds, true, nil, true) {
		if strings.Contains(strings.ToLower(r.Input), "address scope") {
			t.Errorf("%q is a row of its own, and ADR-0079 rules the address scope a lever on the gate", r.Input)
		}
		if r.Input == custodyGateInput {
			gates++
		}
	}
	if gates != 1 {
		t.Fatalf("the ledger holds %d custody-gate rows, want 1", gates)
	}
	if detail := custodyGateRowOf(t, addressScopeSeeds(t)...).StateDetail; !strings.Contains(detail, "address scope") {
		t.Errorf("the detail never names the address scope lever; got %q", detail)
	}
}

// Two rows name the custody extension, and neither may claim the other's figure (#1922).

func TestNeitherExtensionRowClaimsTheOthersFigure(t *testing.T) {
	seeds := []db.ListSeedsRow{nameSeed(false)}
	rows := apertureStatement(nil, true, seeds, testDNSCadenceSeconds, true, nil, true)
	gate := statementRow(t, rows, custodyGateInput)
	ports := statementRow(t, rows, portTierInput)
	sensitive := strconv.Itoa(vergecore.Default().Count().Sensitive)

	for _, cell := range []string{gate.State, gate.StateDetail, gate.RemedyWhy} {
		for _, banned := range []string{"sensitive pair", sensitive} {
			if strings.Contains(cell, banned) {
				t.Errorf("the custody row claims the port-tier row's figure %q; got %q", banned, cell)
			}
		}
	}
	for _, cell := range []string{ports.State, ports.StateDetail, ports.RemedyWhy} {
		if strings.Contains(cell, "name scope extended") || strings.Contains(cell, "name scopes extended") {
			t.Errorf("the port-tier row claims the custody row's scope count; got %q", cell)
		}
	}
	for _, f := range ports.Figures {
		if strings.Contains(f.Text, "name scope") {
			t.Errorf("a port-tier figure counts name scopes; got %q", f.Text)
		}
	}
	if len(gate.Figures) != 1 {
		t.Fatalf("figures = %+v, want the single scope count", gate.Figures)
	}
}
