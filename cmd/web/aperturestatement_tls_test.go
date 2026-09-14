package main

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
	"github.com/winniel123/verge-asm/internal/scan"
)

func tlsCandidateRowOf(t *testing.T) apertureRowView {
	t.Helper()
	return statementRow(t, apertureStatement(nil, true, nameOnlySeeds(), testDNSCadenceSeconds, true, nil, true), tlsCandidateInput)
}

// A typed set stops agreeing with the wire the day the offer gains a version (SPEC §6.1).

func TestTLSRowStatesTheDeclaredOffer(t *testing.T) {
	versions, ciphers := tlsoffer.Versions(), tlsoffer.Ciphers()
	if len(versions) == 0 || len(ciphers) == 0 {
		t.Fatal("the offer declares no version or no suite, so the row has nothing to state")
	}

	row := tlsCandidateRowOf(t)
	for _, v := range versions {
		if !strings.Contains(row.State, v) {
			t.Errorf("state = %q, which drops the declared version %q", row.State, v)
		}
	}
	// ADR-0025 widened the floor to read a TLS-1.0-only listener, so the chip must show it.
	if !strings.Contains(row.State, "TLS "+tlsoffer.TLS10) {
		t.Errorf("state = %q, which never names the TLS 1.0 floor", row.State)
	}
	if !strings.Contains(row.State, strconv.Itoa(len(ciphers))+" cipher suites") {
		t.Errorf("state = %q, want the declared suite count %d", row.State, len(ciphers))
	}
}

// A spelled count outlives the list it counts, and no gate reads copy (SPEC §6.1).

func TestTLSRowTypesNoCountItCannotDerive(t *testing.T) {
	sizes := []string{"nineteen", "four", strconv.Itoa(len(tlsoffer.Ciphers())), strconv.Itoa(len(tlsoffer.Versions()))}
	// A bare substring reads the 3 of `TLS 1.3` as a count, so each size matches as a word.
	word := regexp.MustCompile(`(?i)\b(` + strings.Join(sizes, "|") + `)\b`)

	row := tlsCandidateRowOf(t)
	for _, cell := range []string{row.CadenceWhy, row.StateDetail, row.RemedyWhy} {
		if m := word.FindString(cell); m != "" {
			t.Errorf("the copy types the offer's size %q, so it outlives the offer; got %q", m, cell)
		}
	}
}

// One tlsoffer package feeds two exchanges, so the cell owes both carriers (#1883 addendum).

func TestTLSRowCadenceNamesBothCarriers(t *testing.T) {
	row := tlsCandidateRowOf(t)
	for _, want := range []string{"weekly", "daily", "monthly"} {
		if !strings.Contains(row.Cadence, want) {
			t.Errorf("cadence = %q, which drops the %q edge", row.Cadence, want)
		}
	}
	for _, want := range []string{"tls-acceptance", "certificate"} {
		if !strings.Contains(row.CadenceWhy, want) {
			t.Errorf("the cadence reason never names the %s carrier; got %q", want, row.CadenceWhy)
		}
	}
	// No tls-acceptance setter ships in db/queries, so a dial claim here would be false (#1883).
	if !strings.Contains(strings.ToLower(row.CadenceWhy), "release-coupled") {
		t.Errorf("the cadence reason never says neither cadence is an operator dial; got %q", row.CadenceWhy)
	}
}

// An offer the operator can narrow is a finding the operator can silence (ADR-0030).

func TestTLSRowOffersNoRemedyAndSaysWhy(t *testing.T) {
	row := tlsCandidateRowOf(t)
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

// No toggle narrows this input, so the chip must not borrow an on/off it does not have.

func TestTLSRowStateCarriesNoToggle(t *testing.T) {
	if kind := tlsCandidateRowOf(t).StateKind; kind != "fixed" {
		t.Errorf("state_kind = %q, want `fixed` — the chip takes no status dot", kind)
	}
}

// Every line reads declared configuration and no line reads a batch (SPEC §2.4, §8.8).

func TestTLSRowIsConstantAcrossEstates(t *testing.T) {
	bare := statementRow(t, apertureStatement(nil, true, nil, testDNSCadenceSeconds, true, nil, true), tlsCandidateInput)
	peopled := statementRow(t, apertureStatement(
		[]db.SourceState{{Slug: scan.CTTailSource, Enabled: true}}, true,
		addressScopeSeeds(t), 7*86400, true,
		[]custody.VantageClass{custody.ClassInternet, custody.ClassInternal}, true,
	), tlsCandidateInput)

	if !reflect.DeepEqual(bare, peopled) {
		t.Errorf("the row moved with the estate, so it reads more than declared configuration:\n bare = %+v\n peopled = %+v", bare, peopled)
	}
	// A shipped offerability gate holds every figure here at its own maximum (SPEC §6.3).
	if len(bare.Figures) != 0 {
		t.Errorf("figures = %+v; a figure that fails on nothing inverts the list-movement gate", bare.Figures)
	}
	for _, cell := range []string{bare.Input, bare.Cadence, bare.CadenceWhy, bare.State, bare.StateDetail, bare.Remedy, bare.RemedyWhy} {
		if strings.TrimSpace(cell) == "" {
			t.Error("a cell renders blank, and SPEC §2.3 bars that")
		}
	}
}
