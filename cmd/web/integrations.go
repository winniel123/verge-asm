package main

import (
	"net/http"
	"strings"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Integrations sub-tab's view layer is the design-owned settings.tmpl (its
// "settings-integrations" define, package v3.13.0): the spec catalogue, the PRG
// category/search filter, the spec drawer, and the install/remove/test acts. The
// repo authors no markup here; the handler below wires the authored catalogue and
// the operator's real install state into the tmpl's declared holes (#26j).

// integrationsEnabled gates the whole Settings → Integrations surface (#388). The
// design package is normative for look AND functionality (ADR-0116, PARITY-CHART
// P1.9): the shell's command palette and the Settings tab bar both reach the
// Integrations surface in the design, so it ships enabled rather than hidden. The
// tab, the render dispatch, and the install/disconnect routes are all guarded on
// this one flag, so flipping it true revives the whole surface at once.
//
// "Installing" an integration writes a (slug, "installed") row to integration_state
// and records the operator's declared consent; the real delivery worker
// (internal/delivery/runner.go) POSTs raw JSON to channel URLs and is
// integration-agnostic, so per-integration clients, credential storage, and
// message formatting remain future work layered on top of this surface, not a
// precondition for reaching it. The catalogue, templates, handlers, table
// (db/migrations/21200_integration_state.sql), and queries are all in the tree.
const integrationsEnabled = true

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

// integrationTile is one catalogue row merged with its per-install state, shaped
// for the spec tile grid (#26j). ID is the tile slug; State is the tmpl's display
// vocabulary — installed / attention / available — mapped from the store's
// installed / needs-config / (absent) states (needs-config reads as "needs
// attention" on the spec surface).
type integrationTile struct {
	ID          string
	Name        string
	Mark        string
	Category    string
	Description string
	State       string
}

// integrationDrawerView is the spec drawer's shape (#26j): the tile facts plus the
// grants list, the attention callout, and the installed/last-delivery/classes KV.
// Attention/Installed/LastDelivery/Classes have no read on the live surface, so
// they render empty (their regions collapse) and the design fixture pins them.
type integrationDrawerView struct {
	ID           string
	Name         string
	Mark         string
	Category     string
	Description  string
	State        string
	Attention    string
	Grants       []integrationGrant
	Installed    string
	LastDelivery string
	Classes      string
}

// tileState maps the store's install state to the tmpl's display vocabulary: an
// installed row is "installed"; a needs-config row reads as "attention"; anything
// else (no row) is "available".
func tileState(storeState string) string {
	switch storeState {
	case integrationInstalled:
		return "installed"
	case integrationNeedsConfig:
		return "attention"
	default:
		return "available"
	}
}

// fillIntegrationsSection renders the spec integrations catalogue (#26j): the
// authored catalogue merged with the operator's real install state, filtered by the
// category segment (?cat=) and the search box (?q=), and the spec drawer resolved
// from ?view=. No install state is fabricated — an integration with no stored row is
// available.
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

	tiles := make([]integrationTile, 0, len(integrationCatalog))
	for _, c := range integrationCatalog {
		st := tileState(state[c.Slug])
		if cat != integrationCatAll && c.Category != cat {
			continue
		}
		if ql != "" && !strings.Contains(strings.ToLower(c.Name), ql) && !strings.Contains(strings.ToLower(c.Description), ql) {
			continue
		}
		tiles = append(tiles, integrationTile{
			ID: c.Slug, Name: c.Name, Mark: c.Mark, Category: c.Category,
			Description: c.Description, State: st,
		})
	}

	data["Integrations"] = tiles
	data["IntCats"] = integrationCats
	data["IntCat"] = cat
	data["IntQ"] = q

	// The spec drawer (?view=<id>): any catalogue tile may be opened to read its
	// grants and facts, installed or not.
	if view := r.URL.Query().Get("view"); view != "" {
		if c, ok := integrationBySlug(view); ok {
			data["IntDrawer"] = integrationDrawerView{
				ID: c.Slug, Name: c.Name, Mark: c.Mark, Category: c.Category,
				Description: c.Description, State: tileState(state[c.Slug]),
				Grants: c.Grants,
			}
		}
	}
	return nil
}

// installIntegration records the operator's consent to install a third-party
// integration (#26j). Reaching the install button means the grants have been shown,
// so the click is the consent — grants are all-or-nothing. It is an admin act
// (requireAdmin); an unknown id is refused rather than written. The spec form posts
// an `id`; the pre-spec forms posted `slug`, so both are accepted.
func (s *server) installIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	if _, ok := integrationBySlug(id); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	if _, err := s.store.UpsertIntegrationState(r.Context(), db.UpsertIntegrationStateParams{
		Slug: id, State: integrationInstalled,
	}); err != nil {
		s.serverError(w, "install integration", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=integrations", http.StatusSeeOther)
}

// removeIntegration returns an integration to available (not installed) — the spec
// drawer's Remove act (#26j), and the alias for the pre-spec /disconnect route. It
// is an admin act; an unknown id is refused. Nothing is deleted on the integration's
// own side: this only forgets the local install.
func (s *server) removeIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	if _, ok := integrationBySlug(id); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteIntegrationState(r.Context(), id); err != nil {
		s.serverError(w, "remove integration", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=integrations", http.StatusSeeOther)
}

// testIntegration acknowledges a "Send test" from the spec drawer (#26j). The real
// delivery worker is integration-agnostic (raw JSON to channel URLs), so there is no
// per-integration test send to make yet; this validates the id and returns to the
// tab rather than fabricating a delivery. It is an admin act.
func (s *server) testIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	if _, ok := integrationBySlug(id); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/settings?tab=integrations", http.StatusSeeOther)
}

// integrationFormID reads the integration slug from an `id` field (the spec forms),
// falling back to `slug` (the pre-spec forms).
func integrationFormID(r *http.Request) string {
	if id := r.FormValue("id"); id != "" {
		return id
	}
	return r.FormValue("slug")
}
