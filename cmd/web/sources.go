package main

import (
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
// consent-bar ruling: crt.sh on (throttled); the keyless RIR org→prefix paths
// (ARIN, AFRINIC, APNIC via CAIDA) on; the operator-accepted registry paths
// (RIPEstat, RIPE Database, APNIC registry, LACNIC registry) off; HackerTarget
// and unauthenticated Cert Spotter excluded on terms.
var sourceCatalog = []catalogSource{
	{
		Slug: "crtsh", Name: "crt.sh",
		Authority: "inferred", Completeness: "corroborative", Consent: consentUnencumbered,
		DefaultOn: true,
		ShipNote:  "Certificate transparency logs. Throttled to 5 req/min; a non-200 yields no observation, never an observation of absence.",
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
	ShipNote     string
	ShowGroups   bool
	MayResolve   []string
	Unresolvable []string
}

// coveragePage is the minimal Coverage stub (§6.3). The full aperture statement
// — one line per aperture input, its cadence and on/off state — is ticket 11 and
// is deliberately not built here. This page exists as the entry point to the
// source-enablement modal (§6.3, §6.4).
func (s *server) coveragePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.render(w, "coverage", map[string]any{
		"Title": "Coverage", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
	})
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

	var shipOn, shipOff, barred []sourceView
	for _, v := range views {
		switch {
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
		"ShipOn": shipOn, "ShipOff": shipOff, "Barred": barred,
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
		kind := "source"
		if c.IsProposer {
			kind = "proposer"
		}
		out = append(out, sourceView{
			Slug: c.Slug, Name: c.Name, KindLabel: kind,
			Authority: c.Authority, Completeness: c.Completeness, Consent: c.Consent,
			Enabled: enabled, Toggleable: !c.Barred, ShipNote: c.ShipNote,
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
	if !ok || c.Barred {
		http.Error(w, "unknown source", http.StatusBadRequest)
		return
	}
	enabled, err := strconv.ParseBool(r.FormValue("enabled"))
	if err != nil {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	if _, err := s.store.UpsertSourceState(r.Context(), db.UpsertSourceStateParams{
		Slug: slug, Enabled: enabled, ToggledBy: acct.ID,
	}); err != nil {
		s.serverError(w, "upsert source state", err)
		return
	}
	http.Redirect(w, r, "/sources", http.StatusSeeOther)
}

const sourceTemplates = `
{{define "coverage"}}{{template "head" .}}
{{template "chrome" .}}
<main>
<div class="microlabel">Derived · coverage</div>
<h1>Coverage</h1>
<p>The aperture statement — what each tier is, its cadence, and whether it is on — is not built yet (ticket 11). This page stands in as the entry point to the controls that feed it.</p>
<div class="section">
<div class="microlabel">Sources</div>
<h2>Source enablement</h2>
<p>Which discovery sources may run, and — for the sources that ship off — what accepting their terms means. Turning a source off is always safe; turning one on never adds to the estate on its own.</p>
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
.modal-backdrop { min-height: 100vh; background: rgba(22,22,15,.32);
  display: flex; align-items: flex-start; justify-content: center; padding: var(--space-6); }
.modal { background: var(--surface); border: 1px solid var(--ink);
  box-shadow: 6px 6px 0 rgba(22,22,15,.1); padding: var(--space-6); width: 100%; max-width: 760px; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start;
  border-bottom: 2px solid var(--ink); padding-bottom: var(--space-4); margin-bottom: var(--space-5); }
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
.modal-foot { border-top: 2px solid var(--ink); padding-top: var(--space-4);
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
