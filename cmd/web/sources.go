package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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

// sourcesModal renders the source-enablement surface (§6.4) as the Settings
// sources sub-tab: the discovery-source catalogue split by the state each source
// ships in, with the two marked consent groups on every operator-accepted source.
// A viewer may read it; only an admin sees a toggle. A source is a discovery
// source — distinct from an Integration (a third-party install tile, on its own
// sub-tab, #308): #281 originally parked this catalogue under the "integrations"
// tab as a stopgap before the real Integrations screen existed; #308 gave
// integrations their own tab and returned this catalogue to its own "sources" tab.
func (s *server) sourcesModal(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: "sources"})
}

// fillSourcesSection buckets the discovery-source catalogue by the state each
// source ships in for the sources sub-tab.
func (s *server) fillSourcesSection(r *http.Request, data map[string]any) error {
	views, err := s.sourceViews(r)
	if err != nil {
		return err
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

	data["ShipOn"] = shipOn
	data["ShipOff"] = shipOff
	data["NotExecuting"] = notExecuting
	data["Barred"] = barred
	return nil
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
{{define "srctoggle"}}<form method="post" action="/sources/toggle" style="display:inline">
<input type="hidden" name="slug" value="{{.Slug}}">
<input type="hidden" name="enabled" value="{{if .Enabled}}false{{else}}true{{end}}">
<button class="{{if .Enabled}}secondary{{end}}" type="submit">{{if .Enabled}}Disable{{else}}Enable{{end}}</button>
</form>{{end}}

{{define "settings-sources"}}
<div class="microlabel">Discovery · sources</div>
<h2>Sources</h2>
<p>Which discovery sources may run. Turning a source off never removes anything you already hold — a source's silence never asserted absence. Turning one on lets it run: a source begins observing, a proposer begins offering proposals you confirm into seeds, and neither adds to the estate on its own.</p>
{{if not .IsAdmin}}<div class="notice">You have read access. Enabling or disabling a source is admin-only.</div>{{end}}

<div class="section">
<div class="microlabel">Ship on by default</div>
<table>
<thead><tr><th>Source</th><th>Kind</th><th>Consent</th><th>Authority</th><th>State</th>{{if .IsAdmin}}<th></th>{{end}}</tr></thead>
<tbody>
{{range .ShipOn}}<tr>
<td><div class="mono">{{.Name}}</div><div class="muted">{{.ShipNote}}</div></td>
<td><span class="badge">{{.KindLabel}}</span></td>
<td><span class="badge">{{.Consent}}</span></td>
<td class="mono">{{if .Authority}}{{.Authority}} · {{.Completeness}}{{else}}—{{end}}</td>
<td>{{if .Enabled}}<span class="badge">on</span>{{else}}<span class="badge off">off</span>{{end}}</td>
{{if $.IsAdmin}}<td>{{template "srctoggle" .}}</td>{{end}}
</tr>{{end}}
</tbody>
</table>
</div>

{{if .NotExecuting}}
<div class="section">
<div class="microlabel">Catalogued — not yet executing</div>
<p>These sources are in the catalogue, but no runner ships for them yet — nothing queries them, so
they observe nothing. There is nothing to enable until a runner exists; leaving one here never adds
to the estate and never asserts absence.</p>
<table>
<thead><tr><th>Source</th><th>Kind</th><th>Authority</th><th>State</th></tr></thead>
<tbody>
{{range .NotExecuting}}<tr>
<td><div class="mono">{{.Name}}</div><div class="muted">{{.ShipNote}}</div></td>
<td><span class="badge">{{.KindLabel}}</span></td>
<td class="mono">{{if .Authority}}{{.Authority}} · {{.Completeness}}{{else}}—{{end}}</td>
<td><span class="badge off">not yet executing</span></td>
</tr>{{end}}
</tbody>
</table>
</div>
{{end}}

<div class="section">
<div class="microlabel">Ship off — accept the terms to enable</div>
<p>Each of these ships off. Enabling it is you making a reading the project declined to make on your behalf, so here is what is unresolved, in two groups.</p>
{{range .ShipOff}}
<div class="section">
<div class="custody-head">
<div>
<div class="mono">{{.Name}}</div>
<div class="muted">{{.ShipNote}}</div>
<div style="margin-top:6px"><span class="badge">{{.KindLabel}}</span> <span class="badge">{{.Consent}}</span> {{if .Enabled}}<span class="badge">on</span>{{else}}<span class="badge off">off</span>{{end}}</div>
</div>
{{if $.IsAdmin}}<div>{{template "srctoggle" .}}</div>{{end}}
</div>
<div class="classes" style="align-items:flex-start">
<div style="flex:1;min-width:220px">
<div class="microlabel">What you may be able to resolve</div>
{{if .MayResolve}}<ul>{{range .MayResolve}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="muted">None — every open question here is one nobody has been able to answer.</p>{{end}}
</div>
<div style="flex:1;min-width:220px">
<div class="microlabel">What nobody has been able to resolve</div>
{{if .Unresolvable}}<ul>{{range .Unresolvable}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="muted">None.</p>{{end}}
</div>
</div>
</div>
{{end}}
</div>

<div class="section">
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
</div>
{{end}}
`
