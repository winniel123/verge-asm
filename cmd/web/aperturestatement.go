package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/winniel123/verge-asm/internal/custody"
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

func apertureStatement(seeds []db.ListSeedsRow, classes []custody.VantageClass, classesRead bool) []apertureRowView {
	return []apertureRowView{portTierRow(seeds), vantageClassRow(classes, classesRead)}
}

type apertureVantageStore interface {
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
}

// A failed read renders as withheld, because an empty class set would name a missing leg (#989).

func (s *server) apertureVantageClasses(ctx context.Context, store apertureVantageStore, where string) ([]custody.VantageClass, bool) {
	rows, err := store.ListVantages(ctx)
	if err != nil {
		log.Printf("web: %s: list vantages: %v", where, err)
		return nil, false
	}
	covered, cerr := s.addressScopeCovered(ctx)
	if cerr != nil {
		log.Printf("web: %s: address scope coverage: %v", where, cerr)
		return nil, false
	}
	return listedVantageClasses(rows, covered), true
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

const apertureVantagesHref = "/settings?tab=vantages"

func vantageClassRow(classes []custody.VantageClass, classesRead bool) apertureRowView {
	row := apertureRowView{
		Input: "Vantage class",
		// A value derived at the point of use is never stale, so it has none (ADR-0028, #1896).
		Cadence:    apertureNone,
		CadenceWhy: "The class is derived where it is used, so it carries no currency and needs no cadence.",
		StateKind:  "off",
	}
	if !classesRead {
		row.State = "not read"
		row.StateDetail = "The vantage list did not read, so this cell states no class."
		row.Remedy = apertureNone
		row.RemedyWhy = "A read that did not land names no missing class, so no act follows it."
		return row
	}

	row.State = vantageClassSet(classes)
	row.StateDetail = "A class is derived from the addresses a vantage presents and your declared address scopes, never from a stored field."
	if row.State == "" {
		row.State = apertureNone
		row.StateDetail = "No prober is provisioned, so no vantage presents an address to classify."
	}

	var hasInternet, hasInternal bool
	// Each leg is an existential over the derived class, never a second predicate (#711).
	for _, c := range classes {
		hasInternet = hasInternet || c == custody.ClassInternet
		hasInternal = hasInternal || c == custody.ClassInternal
	}
	row.Remedy, row.RemedyHref = "Provision a prober", apertureVantagesHref
	switch {
	case hasInternet && hasInternal:
		row.StateKind = "on"
		row.Remedy, row.RemedyHref = apertureNone, ""
		row.RemedyWhy = "A vantage reads from each side of your boundary, so no class is missing."
	case hasInternet:
		// The egress step flips an internet prober, so the label names the first act (#1903).
		row.Remedy = "Provision a prober inside your estate"
		row.RemedyWhy = "No declared address scope covers any prober, so no vantage starts the internal leg. Run a prober inside your estate, then declare its egress as an address scope."
	case hasInternal:
		row.RemedyWhy = "No vantage presents an address outside your declared scopes, so no vantage starts the internet leg. Exposure needs an outside observer, unconditionally."
	default:
		row.RemedyWhy = "No vantage presents an observed address, so neither leg has a reader. Exposure needs an outside observer first."
	}
	return row
}

// The count is over our own list of declared vantages, which SPEC §8.8's estate bar does not reach.

func vantageClassSet(classes []custody.VantageClass) string {
	counts := map[string]int{}
	for _, c := range classes {
		counts[string(c)]++
	}
	parts := make([]string, 0, len(counts))
	for _, name := range vantageClassOrder(counts) {
		parts = append(parts, fmt.Sprintf("%d %s", counts[name], name))
	}
	return strings.Join(parts, " · ")
}

// A class the derivation gains renders last rather than dropping out of the count.

func vantageClassOrder(counts map[string]int) []string {
	rest := make(map[string]struct{}, len(counts))
	for name := range counts {
		rest[name] = struct{}{}
	}
	out := make([]string, 0, len(counts))
	for _, c := range []custody.VantageClass{custody.ClassInternet, custody.ClassInternal, custody.ClassUnverified} {
		if counts[string(c)] > 0 {
			out = append(out, string(c))
		}
		delete(rest, string(c))
	}
	return append(out, sortedKeys(rest)...)
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
