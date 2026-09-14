package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/tlsoffer"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/scan"
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

func apertureStatement(states []db.SourceState, statesRead bool, seeds []db.ListSeedsRow, dnsCadence int64, dnsCadenceRead bool, classes []custody.VantageClass, classesRead bool) []apertureRowView {
	return []apertureRowView{
		enabledSourcesRow(states, statesRead),
		portTierRow(seeds),
		custodyGateRow(seeds),
		queriedQtypeSetRow(dnsCadence, dnsCadenceRead),
		tlsCandidateSetRow(),
		vantageClassRow(classes, classesRead),
		controlProbePopulationRow(seeds, dnsCadence, dnsCadenceRead),
	}
}

type apertureSourceStore interface {
	ListSourceStates(ctx context.Context) ([]db.SourceState, error)
}

// A failed read renders as withheld, because an empty override set would name the defaults (#989).

func apertureSourceStates(ctx context.Context, store apertureSourceStore, where string) ([]db.SourceState, bool) {
	rows, err := store.ListSourceStates(ctx)
	if err != nil {
		log.Printf("web: %s: list source states: %v", where, err)
		return nil, false
	}
	return rows, true
}

type apertureCadenceStore interface {
	GetDnsCadenceSeconds(ctx context.Context) (int64, error)
}

// A failed read names no cadence: the default would name a dial the operator moved (#989).

func apertureDNSCadence(ctx context.Context, store apertureCadenceStore, where string) (int64, bool) {
	seconds, err := store.GetDnsCadenceSeconds(ctx)
	if err != nil {
		log.Printf("web: %s: get dns cadence: %v", where, err)
		return 0, false
	}
	if seconds <= 0 {
		// The scan table's CHECK refuses this, so a read that carries it came from elsewhere.
		log.Printf("web: %s: dns cadence is %d seconds, which names no interval", where, seconds)
		return 0, false
	}
	return seconds, true
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
		// A tag failing closed to internet mislabels one vantage; here it would hide the remedy.
		return nil, false
	}
	return listedVantageClasses(rows, covered), true
}

const apertureSourcesHref = "/settings?tab=sources"

// Selection is config-time by key presence, so a declared source is not a source that ran (#1520).

const apertureSourceDetail = "Your own toggles over the shipped defaults, never a batch. " +
	"The count is over sources that admit a Name, so it counts no proposer. " +
	"Only the worker key selects Cert Spotter in place of crt.sh, so this row names what is declared, never what ran."

// A proposer admits no Name, and a bar outranks every toggle, so neither is a source (ADR-0012).

func apertureToggleableSources() []catalogSource {
	out := make([]catalogSource, 0, len(sourceCatalog))
	for _, c := range sourceCatalog {
		if c.IsProposer || c.Barred || c.NoRunner {
			continue
		}
		out = append(out, c)
	}
	return out
}

func enabledSourcesRow(states []db.SourceState, statesRead bool) apertureRowView {
	row := apertureRowView{
		Input:      "Enabled sources",
		Cadence:    "daily · every 5 minutes",
		CadenceWhy: "The ct Scan asks daily and the ct-tail Scan every 5 minutes. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
		StateKind:  "off",
	}
	if !statesRead {
		// A client reads the kind rather than the prose, so `off` would state a state we lack.
		row.StateKind = "withheld"
		row.State = "not read"
		row.StateDetail = "Your source overrides did not read, so this cell names no source."
		row.Remedy = apertureNone
		row.RemedyWhy = "A read that did not land names no source still off, so no act follows it."
		return row
	}

	override := sourceOverrides(states)
	var on, off []string
	var crtshOn, spotterOn bool
	for _, c := range apertureToggleableSources() {
		if !sourceEnabledState(c, override) {
			off = append(off, c.Name)
			continue
		}
		on = append(on, c.Name)
		crtshOn = crtshOn || c.Slug == scan.CrtshSource
		spotterOn = spotterOn || c.Slug == scan.CertSpotterSource
	}

	row.State = strings.Join(on, " · ")
	row.StateDetail = apertureSourceDetail
	if len(on) > 0 {
		row.StateKind = "on"
	} else {
		row.State = apertureNone
		row.StateDetail = "Every source in this count is switched off. " + apertureSourceDetail
	}
	// A zero here is the worst state the row reaches, so it never takes the muted styling.
	row.Figures = []apertureFigureView{{Text: fmt.Sprintf("%d of %d sources enabled", len(on), len(on)+len(off))}}

	if len(off) == 0 {
		row.Remedy = apertureNone
		// A pointer at a screen holding no relevant control is #1854's silence in a new costume.
		row.RemedyWhy = "Every source that admits a Name is enabled. This row counts no proposer and no source barred on terms, because neither admits a Name."
		return row
	}
	row.Remedy, row.RemedyHref = "Enable a source", apertureSourcesHref
	row.RemedyWhy = fmt.Sprintf("A toggle on the Sources tab reaches each source still off: %s.", strings.Join(off, " · "))
	if !spotterOn {
		// The toggle alone never selects it, so a remedy that stops at the tab is unreachable.
		row.RemedyWhy += " Cert Spotter also needs its key on the worker."
	}
	return row
}

const apertureScopeHref = "/scope"

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
		RemedyHref:  apertureScopeHref,
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

const custodyGateDetail = "The gate derives custody before any vantage class is read, and refuses every address it does not derive as yours. " +
	"A declared address scope admits an address directly. " +
	"A custody extension admits the addresses a name scope resolves into."

// The address scope is a lever on this gate, never a row of its own (ADR-0079, #1906).

func custodyGateRow(seeds []db.ListSeedsRow) apertureRowView {
	row := apertureRowView{
		Input:       "The custody gate",
		Cadence:     "every dispatch · daily",
		CadenceWhy:  "The gate runs at every connect dispatch, and the extension's fan-out test rides the daily edge-fanout Scan. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
		StateDetail: custodyGateDetail,
		StateKind:   "off",
		State:       "total · extension off",
		Remedy:      "Extend custody to a name scope",
		RemedyHref:  apertureScopeHref,
	}

	names, extended := 0, 0
	for _, sd := range seeds {
		if sd.Kind != "name" {
			continue
		}
		names++
		if sd.CustodyExtension {
			extended++
		}
	}

	if names == 0 {
		// The extension is barred from an address scope, so a name scope comes first (ADR-0013).
		row.Remedy = "Declare a name scope"
		row.RemedyWhy = "A custody extension is a property of a name scope. Declare a name scope, then extend custody to the addresses it resolves into."
		return row
	}

	// The count is over our own list of declared name scopes, which SPEC §8.8's bar does not reach.
	row.Figures = []apertureFigureView{{
		Text: fmt.Sprintf("%d of %d name %s extended", extended, names, plural(names, "scope", "scopes")),
	}}

	switch {
	case extended == 0:
		row.RemedyWhy = "No name scope carries the extension, so the gate admits an address only where a declared address scope covers it. A switch on the Scope screen extends custody to the addresses a name scope resolves into."
	case extended < names:
		// The switch is per scope, so a binary chip would read `on` over a scope still unextended.
		row.State = "total · extension partial"
		row.RemedyWhy = "A switch on the Scope screen reaches each name scope that does not carry the extension yet."
	default:
		row.State, row.StateKind = "total · extension on", "on"
		row.Remedy, row.RemedyHref = apertureNone, ""
		// A pointer at a screen holding no relevant control is #1854's silence in a new costume.
		row.RemedyWhy = "Every declared name scope carries the extension, so no further switch widens this gate."
	}
	return row
}

const qtypeSetDetail = "The set a prober puts on the wire, never a library default. " +
	"Each qtype is asked by name and never as ANY, because a server may answer ANY with a subset, and a subset licenses no absence. " +
	"The wildcard control probe runs this same set and mints no second list."

func queriedQtypeSetRow(dnsCadence int64, dnsCadenceRead bool) apertureRowView {
	// The set is the leaf's declared offer, so the row reads it and not the dnsQtypeSet mirror.
	offered := resolutionwalk.DefaultOffers().Qtypes
	names := make([]string, 0, len(offered))
	for _, q := range offered {
		names = append(names, string(q))
	}
	row := apertureRowView{
		Input:   "The queried qtype set",
		Cadence: cadenceLabel(dnsCadence),
		// This row's exchange is the dns Scan, whose cadence is one of two dials (#1883).
		CadenceWhy:  "The dns Scan re-asks the whole set on every run. A cadence dial ships for this Scan: the DNS scan interval on the Scope screen moves it.",
		State:       strings.Join(names, " · "),
		StateDetail: qtypeSetDetail,
		// No toggle narrows an offer, so the chip carries no on and no off (ADR-0030).
		StateKind: "fixed",
		Remedy:    apertureNone,
		RemedyWhy: "No setting narrows this set. An offer the operator can narrow is a finding the operator can silence, so the set moves with a release and never with a switch.",
	}
	if !dnsCadenceRead {
		row.Cadence = "not read"
		row.CadenceWhy = "The dns Scan's interval did not resolve on this load, so this cell names no cadence. The set itself ships with the release and is unchanged."
	}
	return row
}

const tlsCandidateSetDetail = "The versions and suites a prober puts on the wire, never a library default. " +
	"The floor is TLS 1.0 on purpose, because a higher floor reports a TLS-1.0-only listener as no TLS at all. " +
	"No TLS 1.3 suite sits in that count, because the library picks the 1.3 suites itself and reads no declared list there."

// One list carries both TLS exchanges, so the row reads the offer and neither caller (ADR-0030 §3).

func tlsCandidateSetRow() apertureRowView {
	ciphers := tlsoffer.Ciphers()
	return apertureRowView{
		Input: "The TLS candidate set",
		// The certificate handshake rides a port tier, so this input moves on three edges (#1883).
		Cadence:    "weekly · daily · monthly",
		CadenceWhy: "The tls-acceptance Scan re-asks the whole set weekly, and the certificate handshake carries the same list on whichever port tier makes the connect: hot daily, and cold monthly where the cold tier runs. Release-coupled: a cadence dial ships for the dns and zone Scans alone.",
		State:      fmt.Sprintf("TLS %s · %d cipher %s", strings.Join(tlsoffer.Versions(), " · "), len(ciphers), plural(len(ciphers), "suite", "suites")),
		// No toggle narrows an offer, so the chip carries no on and no off (ADR-0030).
		StateKind:   "fixed",
		StateDetail: tlsCandidateSetDetail,
		Remedy:      apertureNone,
		RemedyWhy:   "No setting narrows this set. An offer the operator can narrow is a finding the operator can silence, so the set moves with a release and never with a switch. One list serves both TLS exchanges, so widening it would cost a Break on every acceptance and certificate timeline at once.",
	}
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
		// A client reads the kind rather than the prose, so `off` would state a state we lack.
		row.StateKind = "withheld"
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

// The count is over our own list of provisioned probers, which SPEC §8.8's bar does not reach.

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

// A name is discriminated at its parent, so the population is that parent set (ADR-0066).

const controlProbePurpose = "The population discriminates a wildcard: a name is decided at its parent, never at its own apex."

const controlProbeDetail = controlProbePurpose +
	" A control label is generated under the parent of each name the dns Scan resolves inside a declared name scope. " +
	"The probing gate stops the population at your scope, so a parent above your own apex is never probed. " +
	"A name whose parent went unprobed records a Gap and never a value."

// An empty Seed list is a declared state, so this cell states a population and not a pending read.

const controlProbeEmptyDetail = controlProbePurpose +
	" No name scope is declared, so no name resolves under a parent inside one, and the population is empty."

func controlProbePopulationRow(seeds []db.ListSeedsRow, dnsCadence int64, dnsCadenceRead bool) apertureRowView {
	row := apertureRowView{
		Input:   "The control-probe population",
		Cadence: cadenceLabel(dnsCadence),
		// This row's exchange is the dns Scan, whose cadence is one of two dials (#1883).
		CadenceWhy: "The dns Scan rebuilds the population from its own resolution scope on every run. A cadence dial ships for this Scan: the DNS scan interval on the Scope screen moves how often it is rebuilt. No dial moves what it holds.",
		State:      fmt.Sprintf("derived per batch · %d control labels per parent", wildcarddiscrim.LabelCount),
		// No toggle narrows this population, so the chip carries no on and no off (ADR-0030).
		StateKind:   "fixed",
		StateDetail: controlProbeDetail,
		Remedy:      apertureNone,
		RemedyWhy:   "No setting narrows or widens this population on its own. A control probe the operator can suppress is a wildcard finding the operator can silence, so the population follows the name scopes you declare and moves with no switch of its own.",
	}
	if !declaresNameScope(seeds) {
		row.State = apertureNone
		row.StateDetail = controlProbeEmptyDetail
	}
	if !dnsCadenceRead {
		row.Cadence = "not read"
		row.CadenceWhy = "The dns Scan's interval did not resolve on this load, so this cell names no cadence. The population's construction is unchanged: every batch rebuilds it from its own resolution scope."
	}
	return row
}

// The gate stops the population at the Seed, so a declared name scope is its bound (ADR-0066).

func declaresNameScope(seeds []db.ListSeedsRow) bool {
	for _, sd := range seeds {
		if sd.Kind == "name" {
			return true
		}
	}
	return false
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
