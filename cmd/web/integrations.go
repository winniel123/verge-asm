package main

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// integrationsTemplates adds the Integrations sub-tab markup to the shared
// template set (templates.go). It is the Settings → Integrations screen ported
// from design-system/examples/console/Integrations.jsx (ADR-0110): a tile grid of
// third-party install tiles, each opening a Drawer with a ConsentList, with
// install and confirm-gated disconnect. The reference to tmpl orders its
// initialisation before this one. Parsing the markup here keeps this ticket's
// template in this ticket's file (the convention templates_settings.go names).
var _ = template.Must(tmpl.Parse(integrationsTemplates))

// integrationsEnabled gates the whole Settings → Integrations surface (#388). It
// is false: the surface is a placeholder. "Installing" an integration only writes
// a (slug, "installed") row to integration_state, and nothing — no worker, no
// service, no delivery path — consumes that row (the real delivery worker in
// internal/delivery/runner.go POSTs raw JSON to channel URLs and is
// integration-agnostic). There is no client, auth, credential storage, or
// per-integration formatting anywhere in the tree. Shipping a tile grid that lets
// an operator "install" integrations that silently do nothing is misleading, so
// the surface stays hidden until the real build exists.
//
// The tab, the render dispatch, and the install/disconnect routes are all guarded
// on this flag, so with it false nothing integration-related is reachable and no
// user-facing route can write to integration_state. The catalogue, templates,
// handlers, table (db/migrations/21200_integration_state.sql), and queries are
// kept intact (dormant) rather than deleted: the future real build — real
// clients, auth/OAuth, credential storage, per-integration message formatting,
// acks, state mapping, and a worker path that consumes installed integrations —
// revives this surface by flipping this one constant to true.
const integrationsEnabled = false

// The three install states a tile can be in (design-system Integrations.jsx /
// IntegrationTile.jsx). "available" is the absence of a row — nothing installed;
// the store holds only "installed" or "needs-config", the operator-declared
// states. An integration is a third-party install tile, NEVER a delivery channel
// (which carries messages) and NEVER a discovery source (which observes): the word
// stays distinct (CONTEXT.md, and #308's guardrail).
const (
	integrationAvailable   = "available"
	integrationInstalled   = "installed"
	integrationNeedsConfig = "needs-config"

	integrationCatAll = "All"
)

// integrationGrant is one scope an integration receives at install time, shown in
// the ConsentList. Grants are all-or-nothing — a display, not checkboxes — and a
// write-back grant is louder (design-system ConsentList.jsx). A write-back is a
// proposal, never an act: the estate is never mutated by an integration.
type integrationGrant struct {
	Scope  string
	Detail string
	Write  bool
}

// catalogIntegration is one authored row of the integration catalogue. Like the
// source catalogue (ADR-0003) the catalogue is release data — identity, category,
// description, and the consent grants a tile would receive — the same for every
// install, held in the binary. The only per-install fact is the operator's install
// state, and that is what integration_state holds. Marks are neutral letter marks,
// never fake logos (IntegrationTile.jsx).
type catalogIntegration struct {
	Slug        string
	Name        string
	Mark        string
	Category    string
	Description string
	Grants      []integrationGrant
}

// integrationCatalog is the authored library, ported verbatim from
// examples/console/Integrations.jsx's CATALOG — same names, marks, categories,
// descriptions, and grants. Only the sample per-tile `state` is dropped: install
// state is real operator data merged from integration_state, never fabricated.
var integrationCatalog = []catalogIntegration{
	{
		Slug: "slack", Name: "Slack", Mark: "SL", Category: "Notify",
		Description: "Signals and drift summaries as formatted messages, one channel per class.",
		Grants: []integrationGrant{
			{Scope: "Read signals", Detail: "Message content mirrors the signal drawer: fact, evidence, rule."},
			{Scope: "Read drift summaries", Detail: "Batch-level appeared / withdrawn counts."},
		},
	},
	{
		Slug: "pagerduty", Name: "PagerDuty", Mark: "PD", Category: "Notify",
		Description: "Critical signals open incidents; withdrawn signals resolve them.",
		Grants: []integrationGrant{
			{Scope: "Read signals", Detail: "Critical and high only — routing is by class and severity."},
			{Scope: "Write annotations", Detail: "Incident acknowledgement records an annotation on the signal.", Write: true},
		},
	},
	{
		Slug: "teams", Name: "Microsoft Teams", Mark: "MT", Category: "Notify",
		Description: "Adaptive cards for signals and batch completions.",
		Grants: []integrationGrant{
			{Scope: "Read signals"},
			{Scope: "Read batch results", Detail: "Completion, counts, failures."},
		},
	},
	{
		Slug: "jira", Name: "Jira", Mark: "JI", Category: "Ticketing",
		Description: "One issue per signal span. Closing the span closes the issue — never the reverse.",
		Grants: []integrationGrant{
			{Scope: "Read signals", Detail: "Issue fields mirror the signal; severity maps to priority."},
			{Scope: "Write annotations", Detail: "Issue transitions propose an annotation — an operator confirms it.", Write: true},
		},
	},
	{
		Slug: "linear", Name: "Linear", Mark: "LN", Category: "Ticketing",
		Description: "Signals as issues with severity labels and asset links.",
		Grants: []integrationGrant{
			{Scope: "Read signals"},
		},
	},
	{
		Slug: "splunk", Name: "Splunk", Mark: "SP", Category: "SIEM",
		Description: "Every observation and transition as HEC events, source-typed by class.",
		Grants: []integrationGrant{
			{Scope: "Read observations", Detail: "The full evidence stream, not just signals."},
			{Scope: "Read drift transitions"},
		},
	},
	{
		Slug: "elastic", Name: "Elastic", Mark: "EL", Category: "SIEM",
		Description: "Bulk-indexed observations with ECS field mapping.",
		Grants: []integrationGrant{
			{Scope: "Read observations"},
			{Scope: "Read drift transitions"},
		},
	},
	{
		Slug: "s3", Name: "S3-compatible export", Mark: "S3", Category: "Storage",
		Description: "Nightly NDJSON snapshots of inventory, signals, and coverage to your bucket.",
		Grants: []integrationGrant{
			{Scope: "Read inventory"},
			{Scope: "Read signals"},
			{Scope: "Read coverage facts"},
		},
	},
}

// integrationCats is the category segmented control, ported verbatim from
// Integrations.jsx's CATS.
var integrationCats = []string{integrationCatAll, "Notify", "Ticketing", "SIEM", "Storage"}

func integrationBySlug(slug string) (catalogIntegration, bool) {
	for _, c := range integrationCatalog {
		if c.Slug == slug {
			return c, true
		}
	}
	return catalogIntegration{}, false
}

// integrationView is a catalogue row merged with its per-install state, shaped for
// the tile grid. Connected is the union of installed and needs-config — the two
// states that carry a disconnect. OpenURL opens the tile's Drawer; ConfirmURL
// opens the disconnect ConfirmDialog (a link, so no destruction fires on click).
type integrationView struct {
	Slug        string
	Name        string
	Mark        string
	Category    string
	Description string
	State       string
	Installed   bool
	NeedsConfig bool
	Available   bool
	Connected   bool
	Grants      []integrationGrant
	OpenURL     template.URL
	ConfirmURL  template.URL
}

// integrationCatOption is one segment of the category filter.
type integrationCatOption struct {
	Value  string
	Active bool
	URL    template.URL
}

// intURL builds a Settings → Integrations URL carrying the current filter plus any
// extra key/value pairs. Values are encoded by url.Values, and the result is
// marked template.URL so html/template does not re-escape the query separators
// (interpolating a raw fragment mid-URL would otherwise mangle them).
func intURL(cat, q string, kv ...string) template.URL {
	v := url.Values{}
	v.Set("tab", "integrations")
	if cat != "" && cat != integrationCatAll {
		v.Set("cat", cat)
	}
	if q != "" {
		v.Set("q", q)
	}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return template.URL("/settings?" + v.Encode())
}

// fillIntegrationsSection renders the Integrations tile grid (#308): the authored
// catalogue merged with the operator's real install state, filtered by the
// category segment and the search box. It also resolves the open Drawer and the
// disconnect ConfirmDialog from the query string. No install state is fabricated —
// an integration with no stored row is available (not installed).
func (s *server) fillIntegrationsSection(r *http.Request, data map[string]any) error {
	rows, err := s.store.ListIntegrationStates(r.Context())
	if err != nil {
		return err
	}
	state := make(map[string]string, len(rows))
	for _, row := range rows {
		state[row.Slug] = row.State
	}

	cat := r.URL.Query().Get("cat")
	if cat == "" {
		cat = integrationCatAll
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	ql := strings.ToLower(q)

	views := make([]integrationView, 0, len(integrationCatalog))
	bySlug := make(map[string]integrationView, len(integrationCatalog))
	for _, c := range integrationCatalog {
		st := state[c.Slug]
		if st != integrationInstalled && st != integrationNeedsConfig {
			st = integrationAvailable
		}
		v := integrationView{
			Slug: c.Slug, Name: c.Name, Mark: c.Mark, Category: c.Category,
			Description: c.Description, State: st,
			Installed: st == integrationInstalled, NeedsConfig: st == integrationNeedsConfig,
			Available: st == integrationAvailable, Connected: st != integrationAvailable,
			Grants:     c.Grants,
			OpenURL:    intURL(cat, q, "open", c.Slug),
			ConfirmURL: intURL(cat, q, "confirm", c.Slug),
		}
		bySlug[c.Slug] = v
		if cat != integrationCatAll && c.Category != cat {
			continue
		}
		if ql != "" && !strings.Contains(strings.ToLower(c.Name), ql) && !strings.Contains(strings.ToLower(c.Description), ql) {
			continue
		}
		views = append(views, v)
	}

	cats := make([]integrationCatOption, 0, len(integrationCats))
	for _, name := range integrationCats {
		cats = append(cats, integrationCatOption{Value: name, Active: name == cat, URL: intURL(name, q)})
	}

	data["Integrations"] = views
	data["IntCats"] = cats
	data["IntCat"] = cat
	data["IntQuery"] = q
	data["IntCloseURL"] = intURL(cat, q)

	// The open Drawer: any catalogue tile may be opened to read its consent, even
	// one that is available (not installed).
	if open := r.URL.Query().Get("open"); open != "" {
		if v, ok := bySlug[open]; ok {
			data["IntOpen"] = v
		}
	}
	// The disconnect ConfirmDialog renders only for a connected integration: there
	// is nothing to disconnect on an available one, so a stray confirm param on it
	// is ignored rather than offering a destructive act with no target.
	if confirm := r.URL.Query().Get("confirm"); confirm != "" {
		if v, ok := bySlug[confirm]; ok && v.Connected {
			data["IntConfirm"] = v
		}
	}
	return nil
}

// installIntegration records the operator's consent to install a third-party
// integration (#308). Reaching the install button means the ConsentList's grants
// have been shown, so the click is the consent — grants are all-or-nothing. It is
// an admin act (requireAdmin); an unknown slug is refused rather than written.
func (s *server) installIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.FormValue("slug")
	if _, ok := integrationBySlug(slug); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	if _, err := s.store.UpsertIntegrationState(r.Context(), db.UpsertIntegrationStateParams{
		Slug: slug, State: integrationInstalled,
	}); err != nil {
		s.serverError(w, "install integration", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=integrations", http.StatusSeeOther)
}

// disconnectIntegration returns an integration to available (not installed). It is
// reached only through the ConfirmDialog's confirm button (a POST), never fired on
// the tile click — the tile's Disconnect is a link to the confirm step. It is an
// admin act; an unknown slug is refused. Nothing is deleted on the integration's
// own side: this only forgets the local install.
func (s *server) disconnectIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	slug := r.FormValue("slug")
	if _, ok := integrationBySlug(slug); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteIntegrationState(r.Context(), slug); err != nil {
		s.serverError(w, "disconnect integration", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=integrations", http.StatusSeeOther)
}

const integrationsTemplates = `
{{define "int-state-badge"}}{{if .Installed}}<span class="badge" style="color:var(--ok);border-color:var(--ok-border)">installed</span>{{else if .NeedsConfig}}<span class="badge" style="color:var(--warn);border-color:var(--warn-border)">needs config</span>{{else}}<span class="badge off">available</span>{{end}}{{end}}

{{define "int-consent"}}<ul style="list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:2px">
{{range .Grants}}<li style="display:flex;align-items:flex-start;gap:10px;padding:8px 10px;border-radius:10px;{{if .Write}}background:var(--warn-soft){{end}}">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;border-radius:50%;flex:none;margin-top:1px;{{if .Write}}background:var(--warn-soft);border:1px solid var(--warn-border);color:var(--warn){{else}}background:var(--sunken);border:1px solid var(--hairline);color:var(--body){{end}}">{{if .Write}}<svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>{{else}}<svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>{{end}}</span>
<span style="display:flex;flex-direction:column;gap:2px;min-width:0">
<span style="display:flex;align-items:center;gap:8px;font:500 13px var(--sans);color:var(--ink)">{{.Scope}}{{if .Write}} <span class="badge">writes</span>{{end}}</span>
{{if .Detail}}<span style="font:400 12px/1.55 var(--sans);color:var(--muted)">{{.Detail}}</span>{{end}}
</span>
</li>{{end}}
</ul>{{end}}

{{define "settings-integrations"}}
<div class="microlabel">Delivery · integrations</div>
<div class="rulehead">
<div>
<h2 style="margin:0">Integrations</h2>
<span class="muted" style="font-size:12.5px">One-way where possible — Verge pushes, integrations receive. Write-backs are proposals, never acts.</span>
</div>
<div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap">
<div role="group" aria-label="Filter by category" style="display:inline-flex;gap:2px;padding:2px;background:var(--sunken);border:1px solid var(--hairline);border-radius:var(--r-md)">
{{range .IntCats}}<a href="{{.URL}}" style="display:inline-flex;align-items:center;height:28px;padding:0 12px;border-radius:var(--r-sm);font-family:var(--sans);font-size:12.5px;font-weight:500;text-decoration:none;{{if .Active}}background:var(--surface);color:var(--ink);box-shadow:var(--shadow-xs){{else}}color:var(--muted){{end}}">{{.Value}}</a>{{end}}
</div>
<form method="get" action="/settings" style="margin:0">
<input type="hidden" name="tab" value="integrations">
{{if ne .IntCat "All"}}<input type="hidden" name="cat" value="{{.IntCat}}">{{end}}
<input name="q" value="{{.IntQuery}}" placeholder="Search integrations" autocomplete="off" spellcheck="false" style="width:200px;padding:6px 10px;font-size:12.5px">
</form>
</div>
</div>

<div class="banner info">Channels need no integration — Settings → Channels delivers raw JSON to any URL. Integrations add formatting, acks, and state mapping on top.</div>

{{if .Integrations}}
<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:16px">
{{range .Integrations}}
<a href="{{.OpenURL}}" style="text-align:left;background:var(--surface);border:1px solid var(--hairline);border-radius:var(--r-lg);box-shadow:var(--shadow-sm);padding:16px;display:flex;flex-direction:column;gap:10px;min-width:0;text-decoration:none;color:inherit">
<span style="display:flex;align-items:center;gap:10px;min-width:0">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:34px;height:34px;border-radius:10px;background:var(--sunken);border:1px solid var(--hairline);font:600 12px var(--mono);color:var(--body);flex:none">{{.Mark}}</span>
<span style="font:600 13.5px var(--sans);color:var(--ink);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;flex:1 1 auto;min-width:0">{{.Name}}</span>
<span style="flex:none">{{template "int-state-badge" .}}</span>
</span>
<span style="font:400 12.5px/1.55 var(--sans);color:var(--body)">{{.Description}}</span>
<span style="font:500 10.5px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)">{{.Category}}</span>
</a>
{{end}}
</div>
{{else}}
<div class="emptystate">
<h2>No integrations match</h2>
<p>No integration matches your filter. Clear the search or category to see the full library.</p>
</div>
{{end}}

{{if .IntConfirm}}{{with .IntConfirm}}
<a class="scrim" href="{{$.IntCloseURL}}" aria-label="Cancel"></a>
<div class="dialog-panel" role="dialog" aria-modal="true" aria-label="Disconnect {{.Name}}"
	style="position:fixed;top:12vh;left:50%;transform:translateX(-50%);z-index:42">
<div class="microlabel" style="margin-bottom:8px">Disconnect</div>
<h2 style="margin:0 0 8px">Disconnect {{.Name}}</h2>
<p style="margin:0 0 4px">{{.Name}} stops receiving deliveries.</p>
<p class="muted" style="margin:0">Nothing was deleted on the {{.Category}} side — this only forgets the local install.</p>
<div class="dialog-actions">
<a class="btn ghost" href="{{$.IntCloseURL}}">Cancel</a>
<form method="post" action="/settings/integrations/disconnect" style="margin:0">
<input type="hidden" name="slug" value="{{.Slug}}">
<button class="danger" type="submit">Disconnect</button>
</form>
</div>
</div>
{{end}}{{else if .IntOpen}}{{with .IntOpen}}
<a class="scrim" href="{{$.IntCloseURL}}" aria-label="Close"></a>
<div class="drawer-panel" role="dialog" aria-modal="true" aria-label="{{.Name}}">
<div style="display:flex;align-items:center;gap:10px;margin-bottom:18px">
<span aria-hidden="true" style="display:inline-flex;align-items:center;justify-content:center;width:34px;height:34px;border-radius:10px;background:var(--sunken);border:1px solid var(--hairline);font:600 12px var(--mono);color:var(--body)">{{.Mark}}</span>
<h2 style="margin:0;font-size:16px">{{.Name}}</h2>
<span style="margin-left:auto">{{template "int-state-badge" .}}</span>
</div>
<span style="font:500 10.5px var(--mono);letter-spacing:0.07em;text-transform:uppercase;color:var(--muted)">{{.Category}}</span>
<p style="margin:8px 0 0;font:400 13px/1.6 var(--sans);color:var(--body)">{{.Description}}</p>
{{if .NeedsConfig}}<div class="banner warn" style="margin:16px 0 0">Configuration needed — this integration is installed but not yet configured to deliver. Finish setup, then deliveries resume.</div>{{end}}
<div style="margin-top:18px;display:flex;flex-direction:column;gap:8px">
<span class="microlabel">This integration can</span>
{{template "int-consent" .}}
</div>
{{if .Available}}
<div style="margin-top:18px;border-top:1px solid var(--hairline);padding-top:16px">
<p class="muted" style="margin:0 0 12px;font-size:12.5px">Installing grants the access above. Grants are all-or-nothing — there is no partial consent.</p>
<form method="post" action="/settings/integrations/install" style="margin:0">
<input type="hidden" name="slug" value="{{.Slug}}">
<button type="submit" style="width:100%">Install {{.Name}}</button>
</form>
</div>
{{else}}
<div class="drawer-actions">
<a class="btn ghost" href="{{.ConfirmURL}}">Disconnect</a>
<a class="btn secondary" href="{{$.IntCloseURL}}">Close</a>
</div>
{{end}}
</div>
{{end}}{{end}}
{{end}}
`
