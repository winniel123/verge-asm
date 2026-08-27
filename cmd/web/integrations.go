package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/delivery"
	"github.com/winniel123/verge-asm/internal/message"
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
// BoundChannel (#39b) is the id of the delivery Channel this integration is bound to,
// stringified to match an IntChannels option Value — empty when unbound, which the
// drawer renders as "Not connected" and gates "Send test" off.
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
	BoundChannel string
}

// integrationChannelOption is one entry of the drawer's "Delivery channel" select
// (#39b): the tmpl's .IntChannels[{Value,Label,Hint}] hole. Value is the Channel's id
// (stringified) — or "" for the leading "Not connected" entry; Label is the Channel's
// host+path (the fixtures' scheme-stripped form); Hint is a one-line descriptor. The
// binding is a reference to a Channel, never a fold — an integration formats on top of
// a Channel's transport, it is not a Channel itself.
type integrationChannelOption struct {
	Value string
	Label string
	Hint  string
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
	// bound holds each installed integration's bound delivery Channel id, where one is
	// set (a reference to a Channel — #39b). It fills the drawer's BoundChannel hole.
	bound := make(map[string]int64, len(rows))
	for _, row := range rows {
		state[row.Slug] = row.State
		if row.ChannelID.Valid {
			bound[row.Slug] = row.ChannelID.Int64
		}
	}

	// The drawer's "Delivery channel" select options (#39b): the declared Channels, each
	// by its id/host, behind a leading "Not connected" entry (the unbound choice). A
	// Channel list read failure degrades to just "Not connected" — the select still works
	// and an integration can be unbound — rather than 500ing the whole Settings page.
	data["IntChannels"] = s.integrationChannelOptions(r.Context())

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
			dv := integrationDrawerView{
				ID: c.Slug, Name: c.Name, Mark: c.Mark, Category: c.Category,
				Description: c.Description, State: tileState(state[c.Slug]),
				Grants: c.Grants,
			}
			// The bound Channel (#39b): its id stringified to match an IntChannels option
			// Value, so the select renders it selected. Unbound stays "" (Not connected).
			if id, ok := bound[c.Slug]; ok {
				dv.BoundChannel = strconv.FormatInt(id, 10)
			}
			data["IntDrawer"] = dv
		}
	}
	return nil
}

// integrationChannelOptions builds the drawer's "Delivery channel" select (#39b): a
// leading "Not connected" entry (Value "", the unbound choice) followed by one entry
// per declared Channel — Value is the Channel id (stringified), Label its host+path,
// Hint a fixed descriptor. A Channel-list read failure degrades to the "Not connected"
// entry alone, matching the other best-effort reads on this admin page.
func (s *server) integrationChannelOptions(ctx context.Context) []integrationChannelOption {
	opts := []integrationChannelOption{{Value: "", Label: "Not connected", Hint: "no delivery target"}}
	channels, err := s.store.ListChannels(ctx)
	if err != nil {
		log.Printf("web: integrations: list channels for delivery select: %v", err)
		return opts
	}
	for _, c := range channels {
		opts = append(opts, integrationChannelOption{
			Value: strconv.FormatInt(c.ID, 10),
			Label: channelDeliveryLabel(c.Url),
			Hint:  "signed HTTPS channel",
		})
	}
	return opts
}

// channelDeliveryLabel renders a Channel's URL as the select label the fixtures pin:
// the host and path with the scheme stripped (https://ops.acmecorp.io/hook →
// ops.acmecorp.io/hook). A URL that does not parse falls back to its raw form.
func channelDeliveryLabel(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.TrimSuffix(u.Host+u.Path, "/")
	}
	return raw
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

// bindIntegrationChannel binds an installed integration to a delivery Channel, or
// clears the binding (#39b). The drawer's "Delivery channel" select posts {id,channel};
// an empty channel unbinds. The slug is validated against the catalogue and the channel
// against the declared Channels — an unknown either is refused rather than written. The
// binding is a REFERENCE to a Channel, never a fold: the integration formats on top of a
// Channel's transport, it is not a Channel itself. It is an admin act; on success it
// redirects back to the drawer so the operator sees the new binding.
func (s *server) bindIntegrationChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	if _, ok := integrationBySlug(id); !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	dest := "/settings?tab=integrations&view=" + url.QueryEscape(id)

	channel := strings.TrimSpace(r.FormValue("channel"))
	var binding pgtype.Int8
	if channel != "" {
		chID, err := strconv.ParseInt(channel, 10, 64)
		if err != nil {
			http.Error(w, "invalid channel", http.StatusBadRequest)
			return
		}
		// Validate the channel against the declared Channels — a binding to a Channel that
		// does not exist is refused rather than stored (a dangling reference the drawer
		// could not render).
		ok, err := s.channelExists(r.Context(), chID)
		if err != nil {
			s.serverError(w, "bind integration channel: list channels", err)
			return
		}
		if !ok {
			http.Error(w, "unknown channel", http.StatusBadRequest)
			return
		}
		binding = pgtype.Int8{Int64: chID, Valid: true}
	}

	if err := s.store.SetIntegrationChannel(r.Context(), db.SetIntegrationChannelParams{
		Slug: id, ChannelID: binding,
	}); err != nil {
		s.serverError(w, "bind integration channel", err)
		return
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// channelExists reports whether a Channel with the given id is declared.
func (s *server) channelExists(ctx context.Context, id int64) (bool, error) {
	channels, err := s.store.ListChannels(ctx)
	if err != nil {
		return false, err
	}
	for _, c := range channels {
		if c.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// testIntegration sends a real test payload through the integration's bound delivery
// Channel and toasts the outcome (#39b, P0.14 — replacing the dead no-op #26j had). A
// bound integration's test is a plainly-marked test Message built through the delivery
// package's own BuildBody path and POSTed via the shared, SSRF-guarded SendSigned
// transport — the exact path the delivery/report-notify runners use, so the test rides
// the same guard and no second HTTP client exists. A 2xx toasts the spec's "Test message
// sent"; any other outcome toasts an honest non-ok degrade. When unbound the template
// already disables the button; the handler defends that too (nothing to send through →
// a "connect a channel" warn, never a fabricated delivery). It is an admin act.
func (s *server) testIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	integ, ok := integrationBySlug(id)
	if !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	dest := "/settings?tab=integrations&view=" + url.QueryEscape(id)

	// Resolve the bound Channel. Unbound (NULL, or no installed row) → nothing to send.
	binding, err := s.store.GetIntegrationChannel(r.Context(), id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.serverError(w, "test integration: read binding", err)
		return
	}
	if err != nil || !binding.Valid {
		s.toastRedirect(w, r, dest, "warn", "No delivery channel",
			"Connect a channel before sending a test.")
		return
	}

	ch, err := s.store.GetChannelForDelivery(r.Context(), binding.Int64)
	if err != nil {
		// The bound Channel is gone (a race with a delete the ON DELETE SET NULL would
		// normally settle) or the read failed — degrade honestly rather than send nothing
		// into the void.
		s.toastRedirect(w, r, dest, "danger", "Test message not sent",
			"The bound delivery channel is unavailable — reconnect a channel and try again.")
		return
	}

	body, err := delivery.MarshalBody(delivery.BuildBody(delivery.Firing{
		Class:       message.ClassDrift,
		Cause:       message.CauseDrift,
		SubjectKind: "integration-test",
		FiredAt:     integ.Slug,
		Instant:     s.now().UTC(),
		Headline:    "Test message from Verge ASM — delivery check for the " + integ.Name + " integration.",
	}, s.externalURL))
	if err != nil {
		s.serverError(w, "test integration: marshal body", err)
		return
	}
	var secret []byte
	if ch.Secret.Valid {
		secret = []byte(ch.Secret.String)
	}

	statusCode, sendErr := s.channelSender.Send(r.Context(), ch.Url, body, secret)
	if sendErr != nil || !delivery.Delivered(statusCode) {
		s.toastRedirect(w, r, dest, "danger", "Test message not sent",
			"Delivery through "+integ.Name+"'s channel failed — check the channel and try again.")
		return
	}
	s.toastRedirect(w, r, dest, "ok", "Test message sent",
		"Check "+integ.Name+" for the delivery.")
}

// integrationFormID reads the integration slug from an `id` field (the spec forms),
// falling back to `slug` (the pre-spec forms).
func integrationFormID(r *http.Request) string {
	if id := r.FormValue("id"); id != "" {
		return id
	}
	return r.FormValue("slug")
}

// channelTestSender POSTs one signed body through a Channel's transport and returns the
// status code (0 on a transport error or a refused target). It is the "Send test" egress
// seam (#39b): production drives delivery.SendSigned over the hardened redirect-refusing
// client; a test injects a fake Doer so a Send-test assertion never touches the network.
type channelTestSender interface {
	Send(ctx context.Context, targetURL string, body, secret []byte) (int, error)
}

// httpChannelSender is the production sender: it drives delivery.SendSigned over the
// hardened, redirect-refusing HTTP client and the real DNS resolver, so an integration
// test send is SSRF-guarded exactly as the delivery and report-notify runners are — one
// transport, one guard, no second client.
type httpChannelSender struct {
	doer     delivery.Doer
	resolver delivery.Resolver
	now      func() time.Time
}

func newHTTPChannelSender(now func() time.Time) *httpChannelSender {
	return &httpChannelSender{doer: delivery.NewHTTPDoer(), resolver: net.DefaultResolver, now: now}
}

func (h *httpChannelSender) Send(ctx context.Context, targetURL string, body, secret []byte) (int, error) {
	return delivery.SendSigned(ctx, h.doer, h.resolver, targetURL, body, secret, h.now().UTC())
}
