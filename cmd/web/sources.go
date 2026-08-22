package main

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// sourceTemplates adds the Coverage stub and the source-enablement modal to the
// shared template set defined in templates.go. Parsing them here — rather than
// inlining them in that file — keeps this ticket's markup in this ticket's file,
// so templates.go only gains a nav link. The reference to tmpl orders its
// initialisation before this one.
var _ = template.Must(tmpl.Parse(sourceTemplates))

// The three consent tiers a source runs under (v1 spec §3.1, ADR-0003, ADR-0023).
// consent names the door, never who walked through it: the value is authored by
// the project and ships in the release, so it is a constant of the catalogue
// below rather than a per-install fact.
const (
	consentUnencumbered = "unencumbered"
	consentAccepted     = "operator-accepted"
)

// catalogSource is one authored row of the source catalogue. The catalogue is
// release data — identity, the three source properties, and the state a source
// *ships* in — the same for every install (ADR-0003). It is held in the binary,
// never the database: the only per-install fact is the operator's override of a
// shipped default, and that is what source_state holds.
//
// A proposer is deliberately kept in the same catalogue as a source even though
// ADR-0012 rules that a proposer is not a source: the enablement screen governs
// both (§6.4's source-enablement prompt covers the RIR registry paths, which
// propose), and the distinction is carried on the row rather than erased. A
// proposer admits nothing, so only consent applies to it — authority and
// completeness are left empty, never invented.
type catalogSource struct {
	Slug         string
	Name         string // the source's real name — a rendering, never the key
	IsProposer   bool   // ADR-0012: a proposer is not a source; only consent applies
	Authority    string // declared / measured / inferred; "" for a proposer
	Completeness string // enumerable / corroborative; "" for a proposer
	Consent      string // the tier it runs under; "" for a barred source
	DefaultOn    bool   // the state it ships in (§3.1)
	Barred       bool   // excluded on terms — no operator reading consents past it
	NoRunner     bool   // catalogued, but no execution path ships yet (#241) — nothing to enable
	ShipNote     string // project-worded, why it ships in this state; never the source's own terms

	// The two marked groups of the consent prompt (§6.4, ADR-0003 second
	// amendment, #47), rendered for an operator-accepted source at the moment
	// it is enabled. Stated in the project's own words about what is
	// unresolved, never the source's terms, and each group renders even when
	// empty — LACNIC's actionable group is empty by construction.
	MayResolve   []string // what you may be able to resolve — operator-varying questions
	Unresolvable []string // what nobody has been able to resolve — the constant questions
}

// sourceCatalog is the authored set the release ships. Defaults are §3.1's
// consent-bar ruling: the keyless RIR org→prefix paths (ARIN, AFRINIC, APNIC via
// CAIDA) on; the operator-accepted registry paths (RIPEstat, RIPE Database, APNIC
// registry, LACNIC registry) off; HackerTarget and unauthenticated Cert Spotter
// excluded on terms. crt.sh ships on and executing (§3.1, throttled): its runner
// is the ct Scan (ADR-0106), which polls certificate transparency and admits
// Names, so it is a live source again — reversing the not-yet-executing state
// #241 held it in until the runner landed.
var sourceCatalog = []catalogSource{
	{
		Slug: "crtsh", Name: "crt.sh",
		Authority: "inferred", Completeness: "corroborative", Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Certificate transparency logs. Admits the Names a certificate's SAN list carries — authority: inferred — never a wildcard, and observes nothing (ADR-0027). Queried on the ct Scan's daily cadence, throttled to 5 req/min; a failed fetch admits nothing and never an absence.",
	},
	{
		Slug: "arin", Name: "ARIN (entities?fn=)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path. Covers North America.",
	},
	{
		Slug: "afrinic", Name: "AFRINIC (CAIDA ⋈ delegated-stats)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path via CAIDA joined to delegated-stats.",
	},
	{
		Slug: "apnic-caida", Name: "APNIC (CAIDA ⋈ delegated-stats)", IsProposer: true, Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Keyless org→prefix path via CAIDA joined to delegated-stats.",
	},
	{
		Slug: "ripestat", Name: "RIPEstat", IsProposer: true, Consent: consentAccepted,
		ShipNote:     "Ships off. Enabling it is your own acceptance of the source's terms. It proposes address scopes; nothing enters the estate until you confirm a proposal into a seed.",
		MayResolve:   []string{"Whether you resell a service built on the source's data.", "Your own reading of whether writing prefixes to an inventory is re-packaging, and of the purpose list you are bound by."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "ripe-db", Name: "RIPE Database", IsProposer: true, Consent: consentAccepted,
		ShipNote:     "Ships off. Enabling it is your own acceptance of the source's terms. It proposes address scopes; nothing enters the estate until you confirm a proposal into a seed.",
		MayResolve:   []string{"Your own reading of whether inventorying your own estate is a permitted purpose."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "apnic-registry", Name: "APNIC registry", IsProposer: true, Consent: consentAccepted,
		ShipNote:     "Ships off. Enabling it is your own acceptance of the source's terms. It proposes address scopes; nothing enters the estate until you confirm a proposal into a seed.",
		MayResolve:   []string{"Whether you hold, or will seek, the registry's approval.", "Your own reading of the retrieval-system clause's carve-out."},
		Unresolvable: []string{"No reply has ever come, and no record of an approach exists."},
	},
	{
		Slug: "lacnic-registry", Name: "LACNIC registry", IsProposer: true, Consent: consentAccepted,
		ShipNote:     "Ships off. Its terms cannot be retrieved, so enabling it accepts a source whose terms nobody has been able to read.",
		MayResolve:   nil, // empty by construction — the actionable group renders empty here (#47)
		Unresolvable: []string{"Nobody has been able to retrieve these terms."},
	},
	{
		Slug: "hackertarget", Name: "HackerTarget",
		Authority: "measured", Completeness: "corroborative", Barred: true,
		ShipNote: "Excluded on terms. Its terms bar the software's inherent behaviour, which fails regardless of who the operator is — so no operator reading consents past it.",
	},
	{
		Slug: "certspotter", Name: "Cert Spotter (unauthenticated)",
		Authority: "inferred", Completeness: "corroborative", Barred: true,
		ShipNote: "Excluded on terms. Its unauthenticated tier is scoped to personal or evaluation use, which the modal operator is outside.",
	},
}

func catalogBySlug(slug string) (catalogSource, bool) {
	for _, c := range sourceCatalog {
		if c.Slug == slug {
			return c, true
		}
	}
	return catalogSource{}, false
}

// sourceView is a catalogue row merged with its per-install override, shaped for
// rendering. Kind and consent stay visible so a proposer never reads as a source.
type sourceView struct {
	Slug         string
	Name         string
	KindLabel    string // "source" or "proposer"
	Authority    string
	Completeness string
	Consent      string
	Enabled      bool
	Toggleable   bool
	NoRunner     bool // catalogued, no execution path yet (#241)
	ShipNote     string
	ShowGroups   bool
	MayResolve   []string
	Unresolvable []string
}

// dnsQtypeSet is the qtype set the dns Scan puts on the wire, one aperture input
// of the seven (§3.2). It is declared release data — authored by the project and
// shipped in the binary, the same for every install — so it is held here beside
// the source catalogue rather than read from Postgres. The authority is
// resolutionwalk.DefaultOffers().Qtypes; this mirror is asserted equal to it by
// TestDNSQtypeSetMatchesLeaf so the two never drift.
var dnsQtypeSet = []string{"A", "AAAA", "CNAME", "NS", "SOA", "MX", "TXT"}

// apertureLine is one rendered row of the aperture statement (§3.2, §6.3): an
// aperture input, what the tier is, its cadence, and its on/off state — never a
// proportion of the operator's estate. That refused estate-completeness score is
// #28's, and ADR-0095 is the rule this screen keeps: it states what the tier is
// and, where relevant, what the instrument is structurally unable to report,
// never how much of the estate is covered.
type apertureLine struct {
	Input   string // the aperture input's name (§3.2)
	Tier    string // what the tier is — the qtype set, the source set, the scope kinds
	Cadence string // how often the covering Scan asks, or "—" where none does yet
	On      bool   // whether anything drives this input on a default install
	State   string // the on/off rendering: a short honest phrase, never a proportion
	Note    string // one honest clause; never a proportion of the estate
}

// unavailableVantageView is one position the Coverage register reports we
// currently cannot observe from (ADR-0108) — its name, class, and the resolver
// that could not be reached. Rendered so a resolver outage reads as *we could
// not look from here*, never as an empty measurement.
type unavailableVantageView struct {
	Name     string
	Class    string
	Resolver string
}

// checkStep is one step of the day-one checklist (§6.3). The zero-coverage state
// renders these as a rendering, not a wizard: each names a capability and, where
// an act genuinely exists, points at the surface that performs it — adding no
// prompt of its own. Href is empty for a step whose surface does not exist yet
// (running the first batch is the worker's job at cadence, not a button).
type checkStep struct {
	Title string
	Body  string
	Href  string
	CTA   string
}

// coveragePage renders the aperture statement (§6.3): one line per aperture input
// (§3.2) stating what the tier is, its cadence, and whether it is on — never a
// proportion of the operator's estate. It carries a retention stub (the real
// dials are #26/#28/#29), the day-one checklist in the zero-coverage state, and
// the entry point to the source-enablement modal (§6.4). Scoped to what ships so
// far: the dns Scan and source enablement.
func (s *server) coveragePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// Enabled sources — the first aperture input, counted against the toggleable
	// catalogue: a barred source has no instrument to enable, and a source with no
	// runner has no execution path to enable (#241), so neither is a denominator.
	// This counts sources, never the estate.
	views, err := s.sourceViews(r)
	if err != nil {
		s.serverError(w, "list source states", err)
		return
	}
	var srcEnabled, srcToggleable int
	for _, v := range views {
		if !v.Toggleable {
			continue
		}
		srcToggleable++
		if v.Enabled {
			srcEnabled++
		}
	}

	// Queried address scope — from the operator's Seeds. A name scope enumerates
	// nothing (its addresses arrive by resolution); an address scope enumerates.
	seeds, err := s.store.ListSeeds(ctx)
	if err != nil {
		s.serverError(w, "list seeds", err)
		return
	}
	var nameScopes, addrScopes int
	for _, sd := range seeds {
		if sd.Kind == "address" {
			addrScopes++
		} else {
			nameScopes++
		}
	}

	// Vantages — the provisioned probers, plus the shipped resolver-only `local`
	// vantage the dns Scan always resolves through (migration 18800). An
	// internet-class vantage is what opens the Reach and Exposure timelines; a
	// prober is declared internet only once it re-verifies (§4.2), so a freshly
	// provisioned one is still `unverified`.
	vantages, err := s.store.ListVantages(ctx)
	if err != nil {
		s.serverError(w, "list vantages", err)
		return
	}
	provisioned := len(vantages)
	internetVantage := false
	for _, v := range vantages {
		if v.Class == "internet" {
			internetVantage = true
		}
	}

	// The dns Scan carries the cadence for every input it covers (§3.4, ADR-0084).
	// A cadence is not itself an aperture input — a Batch records what it asked,
	// never how often — but it is what the aperture statement states per line.
	dns, err := s.store.GetScanByKind(ctx, "dns")
	if err != nil {
		s.serverError(w, "get dns scan", err)
		return
	}
	dnsCadence := cadenceLabel(dns.CadenceSeconds)
	dnsOn := dns.Enabled

	// Blanket responders (ADR-0104 §4): addresses whose current reachability is a
	// Gap because they answer on every port — a CDN/anycast/proxy edge, not the
	// origin. This is a read surface in the aperture register, never a Transition and
	// never a new message cause: the finding must be surfaced, not silently absorbed
	// into Gaps, or the operator reads *nothing open* where the honest statement is
	// *we cannot see your origin from here*. A best-effort read — a failure degrades
	// to no statement rather than a 500.
	var blanketAddrs []string
	if svc, berr := s.store.ListBlanketedReachServices(ctx); berr == nil {
		seen := map[string]bool{}
		for _, key := range svc {
			if addr, _, _ := splitServiceKey(key); addr != "" && !seen[addr] {
				seen[addr] = true
				blanketAddrs = append(blanketAddrs, addr)
			}
		}
		sort.Strings(blanketAddrs)
	}

	// Unavailable vantages (ADR-0108): positions we currently cannot observe from
	// — a resolver that went unreachable, a prober that keeps failing. This is the
	// loud surface #249 asks for: it names the position, so a failure reads as *we
	// could not look from here* rather than as an empty measurement, and it is
	// distinct from a subject that genuinely has no records. It includes the
	// resolver-only `local` vantage, which the prober list excludes. Best-effort,
	// like the blanket register: a read failure degrades to no statement.
	var unavailableVantages []unavailableVantageView
	if rows, uerr := s.store.ListUnavailableVantages(ctx); uerr == nil {
		for _, v := range rows {
			resolver := v.Resolver
			if resolver == "" {
				resolver = "—"
			}
			unavailableVantages = append(unavailableVantages, unavailableVantageView{
				Name: v.Name, Class: v.Class, Resolver: resolver,
			})
		}
	}

	scopeState := "no scope declared"
	if nameScopes+addrScopes > 0 {
		scopeState = fmt.Sprintf("%d name · %d address", nameScopes, addrScopes)
	}

	vantageState := "resolver only"
	if provisioned > 0 {
		vantageState = fmt.Sprintf("%d prober · resolver", provisioned)
	}
	vantageNote := "The shipped local resolver position, plus any provisioned probers. No internet-class vantage yet, so the Reach and Exposure timelines have not opened."
	if internetVantage {
		vantageNote = "An internet-class vantage is configured — the Reach and Exposure timelines are open."
	}

	lines := []apertureLine{
		{
			Input: "Enabled sources", Tier: fmt.Sprintf("%d of %d sources enabled", srcEnabled, srcToggleable),
			Cadence: "—", On: srcEnabled > 0, State: fmt.Sprintf("%d on", srcEnabled),
			Note: "Discovery sources that may run. Turning one on never adds to the estate on its own; a proposer only offers proposals you confirm into Seeds.",
		},
		{
			Input: "Port sets", Tier: "hot / cold port tiers",
			Cadence: "—", On: false, State: "off",
			Note: "No port Scan ships yet — the dns Scan carries no port list. The hot and cold tiers, and the sensitive-pairs line, land in later tickets.",
		},
		{
			Input: "Vantages", Tier: "network positions",
			Cadence: dnsCadence, On: dnsOn, State: vantageState,
			Note: vantageNote,
		},
		{
			Input: "TLS candidate set", Tier: "versions & ciphers",
			Cadence: "—", On: false, State: "off",
			Note: "No TLS-acceptance Scan ships yet.",
		},
		{
			Input: "Qtype set", Tier: strings.Join(dnsQtypeSet, " · "),
			Cadence: dnsCadence, On: dnsOn, State: onOff(dnsOn),
			Note: "The seven qtypes the dns Scan asks, explicitly and never as ANY.",
		},
		{
			Input: "Control-probe population", Tier: "parents of resolved names",
			Cadence: dnsCadence, On: dnsOn, State: onOff(dnsOn),
			Note: "Generated under a resolved name's parent for wildcard discrimination. Empty until names resolve; it grows with the estate, never ahead of it.",
		},
		{
			Input: "Queried address scope", Tier: "name & address Seeds",
			Cadence: dnsCadence, On: nameScopes+addrScopes > 0, State: scopeState,
			Note: "What the dns Scan queries. A name scope enumerates nothing — its addresses arrive by resolution — while an address scope walks every address it covers.",
		},
	}

	// Zero coverage is the honest day-one shape: nothing declared to look at yet.
	// The checklist is a rendering of the path out of it, not a wizard.
	zeroCoverage := nameScopes+addrScopes == 0
	steps := []checkStep{
		{
			Title: "Declare your domain",
			Body:  "Name a registrable domain as a Seed. Its addresses arrive by measured resolution — a name scope enumerates nothing on its own.",
			Href:  "/seeds", CTA: "Declare a scope →",
		},
		{
			Title: "Upload a zone file",
			Body:  "Supply your zone as a Source. It proposes the names the resolver walk then measures, re-read on its own cadence.",
			Href:  "/seeds", CTA: "Manage Seeds →",
		},
		{
			Title: "Add an internet vantage",
			Body:  "Provision a prober. The act itself declares the vantage sits on the internet and opens the Reach and Exposure timelines.",
			Href:  "/seeds", CTA: "Provision a prober →",
		},
		{
			Title: "Run the first batch",
			Body:  "The dns Scan fans out over every configured vantage at its cadence once a domain and a vantage exist. It runs on its own — there is no button to press.",
			Href:  "", CTA: "",
		},
	}

	s.render(w, "coverage", map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"Lines": lines, "ZeroCoverage": zeroCoverage, "Steps": steps,
		"DNSCadence": dnsCadence, "DNSOn": dnsOn,
		"BlanketAddrs": blanketAddrs, "UnavailableVantages": unavailableVantages,
	})
}

// cadenceLabel renders a Scan's cadence in seconds as the phrase the aperture
// statement states. The common shipped cadences get a word; anything else is
// stated in its plain unit rather than invented into a near-fit.
func cadenceLabel(seconds int64) string {
	switch {
	case seconds <= 0:
		return "—"
	case seconds%86400 == 0:
		if seconds == 86400 {
			return "daily"
		}
		return fmt.Sprintf("every %d days", seconds/86400)
	case seconds%3600 == 0:
		if seconds == 3600 {
			return "hourly"
		}
		return fmt.Sprintf("every %d hours", seconds/3600)
	case seconds%60 == 0:
		return fmt.Sprintf("every %d minutes", seconds/60)
	default:
		return fmt.Sprintf("every %d seconds", seconds)
	}
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}

// sourcesModal renders the source-enablement modal (§6.4): the catalogue split
// by the state each source ships in, with the two marked consent groups on every
// operator-accepted source. A viewer may read it; only an admin sees a toggle.
func (s *server) sourcesModal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	views, err := s.sourceViews(r)
	if err != nil {
		s.serverError(w, "list source states", err)
		return
	}

	var shipOn, shipOff, notExecuting, barred []sourceView
	for _, v := range views {
		// A no-runner source is also non-toggleable, so its case must precede the
		// barred case below — reversing the two would sink it into the barred
		// bucket and render it as excluded-on-terms rather than not-yet-executing.
		switch {
		case v.NoRunner:
			notExecuting = append(notExecuting, v)
		case !v.Toggleable:
			barred = append(barred, v)
		case v.ShowGroups:
			shipOff = append(shipOff, v)
		default:
			shipOn = append(shipOn, v)
		}
	}

	s.render(w, "sources", map[string]any{
		"Title": "Source enablement", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"ShipOn": shipOn, "ShipOff": shipOff, "NotExecuting": notExecuting, "Barred": barred,
	})
}

// sourceViews merges the authored catalogue with the operator's overrides. A
// source's effective state is its override where one exists and its shipped
// default otherwise.
func (s *server) sourceViews(r *http.Request) ([]sourceView, error) {
	states, err := s.store.ListSourceStates(r.Context())
	if err != nil {
		return nil, err
	}
	override := make(map[string]bool, len(states))
	for _, st := range states {
		override[st.Slug] = st.Enabled
	}

	out := make([]sourceView, 0, len(sourceCatalog))
	for _, c := range sourceCatalog {
		enabled := c.DefaultOn
		if o, ok := override[c.Slug]; ok {
			enabled = o
		}
		// A source with no runner has nothing to enable — its effective state is
		// never `on`, whatever an override or default might say — and it is not
		// toggleable, exactly as a barred source is not (#241).
		if c.NoRunner {
			enabled = false
		}
		kind := "source"
		if c.IsProposer {
			kind = "proposer"
		}
		out = append(out, sourceView{
			Slug: c.Slug, Name: c.Name, KindLabel: kind,
			Authority: c.Authority, Completeness: c.Completeness, Consent: c.Consent,
			Enabled: enabled, Toggleable: !c.Barred && !c.NoRunner, NoRunner: c.NoRunner,
			ShipNote:     c.ShipNote,
			ShowGroups:   c.Consent == consentAccepted,
			MayResolve:   c.MayResolve,
			Unresolvable: c.Unresolvable,
		})
	}
	return out, nil
}

// toggleSource records an admin's on/off choice for one source. Toggling is an
// authenticated admin act (it reaches here only through requireAdmin). A barred
// source has no consent instrument the modal operator can satisfy, so it cannot
// be toggled on or off; an unknown slug is refused rather than written.
func (s *server) toggleSource(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.FormValue("slug")
	c, ok := catalogBySlug(slug)
	// A barred source has no consent instrument to satisfy, and a source with no
	// runner (#241) has nothing to run, so neither is toggleable; an unknown slug
	// is refused rather than written.
	if !ok || c.Barred || c.NoRunner {
		http.Error(w, "unknown source", http.StatusBadRequest)
		return
	}
	enabled, err := strconv.ParseBool(r.FormValue("enabled"))
	if err != nil {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: slug, Enabled: enabled,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	http.Redirect(w, r, "/sources", http.StatusSeeOther)
}

const sourceTemplates = `
{{define "coverage"}}{{template "head" .}}
<style>
.aperture td { vertical-align: top; }
.aperture .in { font-weight: 600; white-space: nowrap; }
.aperture .tier { font-family: var(--mono); font-size: 12px; }
.aperture .note { color: var(--muted); font-size: 12px; margin-top: 4px; }
.aperture .cad { font-family: var(--mono); font-size: 12px; color: var(--muted); white-space: nowrap; }
.checklist { list-style: none; margin: 0; padding: 0; counter-reset: step; }
.checklist > li { border: 1px solid var(--hairline); padding: var(--space-4); margin-bottom: var(--space-4);
  display: flex; gap: var(--space-4); align-items: flex-start; }
.checklist .num { counter-increment: step; font-family: var(--mono); font-weight: 600; font-size: 12px;
  border: 1px solid var(--ink); width: 22px; height: 22px; flex: none;
  display: flex; align-items: center; justify-content: center; }
.checklist .num::before { content: counter(step); }
.checklist .step-body { flex: 1; }
.checklist h3 { font-size: 13px; margin: 0 0 4px; }
.checklist p { margin: 0 0 var(--space-3); font-size: 12px; color: var(--muted); }
.checklist .no-surface { font-family: var(--mono); font-size: 11px; color: var(--muted);
  text-transform: uppercase; letter-spacing: 0.06em; }
</style>
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · coverage</div>
<h1>Coverage</h1>
<p>The aperture statement: one line per aperture input, what the tier is, its cadence, and whether
it is on. It states what the instrument looks at — never a proportion of your estate. Scoped to
what ships so far: the dns Scan and source enablement.</p>

<div class="section">
<div class="microlabel">Aperture statement</div>
<h2>Seven aperture inputs</h2>
<table class="aperture">
<thead><tr><th>Input</th><th>Tier</th><th>Cadence</th><th>State</th></tr></thead>
<tbody>
{{range .Lines}}<tr>
<td class="in">{{.Input}}</td>
<td><div class="tier">{{.Tier}}</div><div class="note">{{.Note}}</div></td>
<td class="cad">{{.Cadence}}</td>
<td>{{if .On}}<span class="badge">{{.State}}</span>{{else}}<span class="badge off">{{.State}}</span>{{end}}</td>
</tr>{{end}}
</tbody>
</table>
</div>

{{if .BlanketAddrs}}
<div class="section">
<div class="microlabel">Aperture · blanket responders</div>
<h2>These addresses answer on every port</h2>
<p>{{len .BlanketAddrs}} address{{if ne (len .BlanketAddrs) 1}}es{{end}} in your estate answer TCP on all ports — a
proxy edge (a CDN, anycast front, or reverse proxy), not your origin. We measured this with a
control-port probe, not read it off any provider list. Their reaches are recorded as a Gap, never
<span class="mono">reached</span>: from here we cannot tell a real origin service behind the edge from
the edge answering for it, so their ports do not count as open and no sensitive-port signal fires on
them. To measure the real surface, declare your origin IPs as an address scope.</p>
<table>
<thead><tr><th>Address</th><th>What it is</th></tr></thead>
<tbody>
{{range .BlanketAddrs}}<tr>
<td class="mono">{{.}}</td>
<td class="muted">proxy edge — answers on all ports, origin not visible from here</td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

{{if .UnavailableVantages}}
<div class="section">
<div class="microlabel">Aperture · unavailable vantages</div>
<h2>We cannot look from {{len .UnavailableVantages}} position{{if ne (len .UnavailableVantages) 1}}s{{end}}</h2>
<p>A vantage is a network position we measure from, and its recursive resolver is part of it. These
positions could not be reached — a resolver pointed at nothing, or a prober that kept failing — so
their most recent batches failed and covered nothing. This is <em>we could not look from here</em>,
not <em>we looked and there is nothing there</em>: no empty measurement is committed for them, and the
Reach they would have measured is a Gap rather than a clean empty result. An internet-class position
here means Exposure that needs it is absent, not quietly computed from the class that still answers.</p>
<table>
<thead><tr><th>Vantage</th><th>Class</th><th>Resolver</th><th>State</th></tr></thead>
<tbody>
{{range .UnavailableVantages}}<tr>
<td class="mono">{{.Name}}</td>
<td>{{.Class}}</td>
<td class="mono">{{.Resolver}}</td>
<td><span class="badge off">unavailable</span></td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

{{if .ZeroCoverage}}
<div class="section">
<div class="microlabel">Day one</div>
<h2>Nothing is covered yet</h2>
<p>Nothing is declared to look at, so the aperture reads empty above. Four steps set the estate;
each names a capability and, where an act exists, points at the surface that performs it.</p>
<ol class="checklist">
{{range .Steps}}<li>
<div class="num"></div>
<div class="step-body">
<h3>{{.Title}}</h3>
<p>{{.Body}}</p>
{{if .Href}}<a class="btn" href="{{.Href}}" style="text-decoration:none">{{.CTA}}</a>{{else}}<div class="no-surface">Runs automatically at cadence</div>{{end}}
</div>
</li>{{end}}
</ol>
</div>
{{end}}

<div class="section">
<div class="microlabel">Retention · stub</div>
<h2>Retention dials</h2>
<p>Two dials govern how long the record is kept: the operational Dispatch floor and the
observation-currency floor. The live controls land in later tickets — this section marks their
place.</p>
<div class="kv"><div class="k">Dispatch floor</div><div><span class="badge off">not configurable yet</span></div></div>
<div class="kv"><div class="k">Currency floor</div><div><span class="badge off">not configurable yet</span></div></div>
</div>

<div class="section">
<div class="microlabel">Sources</div>
<h2>Source enablement</h2>
<p>Which discovery sources may run, and — for the sources that ship off — what accepting their terms
means. Turning a source off is always safe; turning one on never adds to the estate on its own.</p>
<a class="btn" href="/sources" style="text-decoration:none">Manage source enablement →</a>
</div>
</main>
{{template "foot" .}}{{end}}

{{define "srctoggle"}}<form method="post" action="/sources/toggle" style="display:inline">
<input type="hidden" name="slug" value="{{.Slug}}">
<input type="hidden" name="enabled" value="{{if .Enabled}}false{{else}}true{{end}}">
<button class="{{if .Enabled}}secondary{{end}}" type="submit">{{if .Enabled}}Disable{{else}}Enable{{end}}</button>
</form>{{end}}

{{define "sources"}}{{template "head" .}}
<style>
.modal-backdrop { min-height: 100vh; background: rgba(21,18,15,0.4);
  display: flex; align-items: flex-start; justify-content: center; padding: var(--space-6); }
.modal { background: var(--surface); border: 1px solid var(--hairline);
  border-radius: var(--r-lg); box-shadow: var(--shadow-sm); padding: var(--space-6); width: 100%; max-width: 760px; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start;
  border-bottom: 1px solid var(--hairline); padding-bottom: var(--space-4); margin-bottom: var(--space-5); }
.modal-head h1 { margin: 0; }
.modal-close { font-family: var(--mono); font-size: 16px; text-decoration: none; color: var(--ink); }
.src-note { color: var(--muted); font-size: 12px; }
.src-block { border: 1px solid var(--hairline); padding: var(--space-4); margin-bottom: var(--space-4); }
.src-block-head { display: flex; justify-content: space-between; align-items: flex-start;
  gap: var(--space-4); margin-bottom: var(--space-4); }
.groups { display: flex; gap: var(--space-5); flex-wrap: wrap; }
.group { flex: 1; min-width: 220px; }
.group ul { margin: 4px 0 0; padding-left: 18px; }
.group li { margin-bottom: 6px; }
.group .empty { color: var(--muted); margin: 4px 0 0; }
.badge.off { color: var(--muted); border-color: var(--hairline); }
.modal-foot { border-top: 1px solid var(--hairline); padding-top: var(--space-4);
  margin-top: var(--space-5); display: flex; justify-content: flex-end; }
</style>
<div class="modal-backdrop"><div class="modal">
<div class="modal-head">
<div><div class="microlabel">Declared · sources</div><h1>Source enablement</h1></div>
<a class="modal-close" href="/coverage" aria-label="Close">✕</a>
</div>

<p>Which discovery sources may run. Turning a source off never removes anything you already hold — a source's silence never asserted absence. Turning one on lets it run: a source begins observing, a proposer begins offering proposals you confirm into seeds, and neither adds to the estate on its own.</p>
{{if not .IsAdmin}}<div class="notice">You have read access. Enabling or disabling a source is admin-only.</div>{{end}}

<div class="microlabel">Ship on by default</div>
<table>
<thead><tr><th>Source</th><th>Kind</th><th>Consent</th><th>Authority</th><th>State</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .ShipOn}}<tr>
<td><div class="mono">{{.Name}}</div><div class="src-note">{{.ShipNote}}</div></td>
<td><span class="badge">{{.KindLabel}}</span></td>
<td><span class="badge">{{.Consent}}</span></td>
<td class="mono">{{if .Authority}}{{.Authority}} · {{.Completeness}}{{else}}—{{end}}</td>
<td>{{if .Enabled}}<span class="badge">on</span>{{else}}<span class="badge off">off</span>{{end}}</td>
{{if $.IsAdmin}}<td>{{template "srctoggle" .}}</td>{{end}}
</tr>{{end}}
</tbody>
</table>

{{if .NotExecuting}}
<div class="microlabel">Catalogued — not yet executing</div>
<p>These sources are in the catalogue, but no runner ships for them yet — nothing queries them, so
they observe nothing. There is nothing to enable until a runner exists; leaving one here never adds
to the estate and never asserts absence.</p>
<table>
<thead><tr><th>Source</th><th>Kind</th><th>Authority</th><th>State</th></tr></thead>
<tbody>
{{range .NotExecuting}}<tr>
<td><div class="mono">{{.Name}}</div><div class="src-note">{{.ShipNote}}</div></td>
<td><span class="badge">{{.KindLabel}}</span></td>
<td class="mono">{{if .Authority}}{{.Authority}} · {{.Completeness}}{{else}}—{{end}}</td>
<td><span class="badge off">not yet executing</span></td>
</tr>{{end}}
</tbody>
</table>
{{end}}

<div class="microlabel">Ship off — accept the terms to enable</div>
<p>Each of these ships off. Enabling it is you making a reading the project declined to make on your behalf, so here is what is unresolved, in two groups.</p>
{{range .ShipOff}}
<div class="src-block">
<div class="src-block-head">
<div>
<div class="mono">{{.Name}}</div>
<div class="src-note">{{.ShipNote}}</div>
<div style="margin-top:6px"><span class="badge">{{.KindLabel}}</span> <span class="badge">{{.Consent}}</span> {{if .Enabled}}<span class="badge">on</span>{{else}}<span class="badge off">off</span>{{end}}</div>
</div>
{{if $.IsAdmin}}<div>{{template "srctoggle" .}}</div>{{end}}
</div>
<div class="groups">
<div class="group">
<div class="microlabel">What you may be able to resolve</div>
{{if .MayResolve}}<ul>{{range .MayResolve}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="empty">None — every open question here is one nobody has been able to answer.</p>{{end}}
</div>
<div class="group">
<div class="microlabel">What nobody has been able to resolve</div>
{{if .Unresolvable}}<ul>{{range .Unresolvable}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="empty">None.</p>{{end}}
</div>
</div>
</div>
{{end}}

<div class="microlabel">Excluded on terms</div>
<table>
<thead><tr><th>Source</th><th>Kind</th><th>Why it stays off</th></tr></thead>
<tbody>
{{range .Barred}}<tr>
<td><div class="mono">{{.Name}}</div></td>
<td><span class="badge">{{.KindLabel}}</span></td>
<td>{{.ShipNote}}</td>
</tr>{{end}}
</tbody>
</table>

<div class="modal-foot"><a class="btn secondary" href="/coverage" style="text-decoration:none">Done</a></div>
</div></div>
{{template "foot" .}}{{end}}
`
