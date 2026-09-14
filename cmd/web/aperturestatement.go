package main

import (
	"fmt"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// An empty cell renders this word plus its reason, so no cell is ever blank (ADR-0044, #1886).

const apertureNone = "none"

type apertureFigureView struct {
	Text string
	Zero bool
}

type apertureRowView struct {
	Input       string
	Cadence     string
	CadenceWhy  string
	State       string
	StateKind   string
	Figures     []apertureFigureView
	StateDetail string
	Remedy      string
	RemedyHref  string
	RemedyWhy   string
}

// The card renders the rows that exist and grows to the seven of the spec's order (#1917).

func apertureStatement(seeds []db.ListSeedsRow) []apertureRowView {
	return []apertureRowView{portTierRow(seeds)}
}

func portTierRow(seeds []db.ListSeedsRow) apertureRowView {
	core := vergecore.Default()
	sensitive := core.Count().Sensitive
	udp := sensitiveUDPPairs(core)

	row := apertureRowView{
		Input:       "Port and transport tiers",
		Cadence:     "daily · monthly",
		CadenceWhy:  "hot daily, cold monthly. Release-coupled: no operator dial exists.",
		State:       "hot on · cold off · udp no flag",
		StateKind:   "off",
		StateDetail: "The cold tier's state is the shadow of an empty scope list, not a switch. UDP has no flag at all.",
		Remedy:      "Declare an address scope",
		RemedyHref:  "/scope",
		RemedyWhy:   "No declared scope reads a sensitive pair. An address scope, or a custody extension on a name scope, moves this figure.",
	}
	unread := sensitive
	if apertureLeverDeclared(seeds) {
		unread = udp
		row.Remedy, row.RemedyHref = apertureNone, ""
		// A pointer at a screen holding no relevant control is #1854's silence in a new costume.
		row.RemedyWhy = fmt.Sprintf("The %d pairs still unread are UDP. No tier reads them, and no setting opens one.", udp)
	}
	row.Figures = []apertureFigureView{
		{Text: fmt.Sprintf("%d of %d sensitive pairs unread", unread, sensitive), Zero: unread == 0},
		// UDP is never probed, so no UDP pair is ever inside the recorded scope (ADR-0083).
		{Text: fmt.Sprintf("0 of %d sensitive pairs the instrument cannot report as reached", sensitive), Zero: true},
		// Configuration absence routes outside the domain, so no rule is unevaluable (ADR-0095).
		{Text: fmt.Sprintf("0 of %d rules unevaluable", len(signal.AllRuleNames())), Zero: true},
	}
	return row
}

// `Counts.UDP` counts the union, not the sensitive half, so the caller filters (#1884).

func sensitiveUDPPairs(core vergecore.List) int {
	n := 0
	for _, p := range core.SensitivePairs() {
		if p.Transport == vergecore.UDP {
			n++
		}
	}
	return n
}

// ADR-0009's union holds every TCP sensitive pair, so either lever leaves only UDP (#1883).

func apertureLeverDeclared(seeds []db.ListSeedsRow) bool {
	for _, sd := range seeds {
		if sd.Kind == "address" && sd.AddressCidr != nil {
			return true
		}
		if sd.CustodyExtension {
			return true
		}
	}
	return false
}
