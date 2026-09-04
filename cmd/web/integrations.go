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

const integrationsEnabled = true

// An integration is neither a Channel nor a Source — it formats on top of one (CONTEXT.md).

const (
	integrationAvailable   = "available"
	integrationInstalled   = "installed"
	integrationNeedsConfig = "needs-config"

	integrationCatAll = "All"
)

type integrationGrant struct {
	Scope  string
	Detail string
	Write  bool // A write-back is a proposal; an integration never mutates the estate.
}

type catalogIntegration struct {
	Slug        string
	Name        string
	Mark        string // A neutral letter mark, never a third-party logo.
	Category    string
	Description string
	Grants      []integrationGrant
}

// A verbatim port of the example's CATALOG, never editorial copy (ADR-0110).

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

var integrationCats = []string{integrationCatAll, "Notify", "Ticketing", "SIEM", "Storage"}

func integrationBySlug(slug string) (catalogIntegration, bool) {
	for _, c := range integrationCatalog {
		if c.Slug == slug {
			return c, true
		}
	}
	return catalogIntegration{}, false
}

type integrationTile struct {
	ID          string
	Name        string
	Mark        string
	Category    string
	Description string
	State       string
}

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

type integrationChannelOption struct {
	Value string
	Label string
	Hint  string
}

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

func (s *server) fillIntegrationsSection(r *http.Request, data map[string]any) error {
	rows, err := s.store.ListIntegrationStates(r.Context())
	if err != nil {
		return err
	}
	state := make(map[string]string, len(rows))
	bound := make(map[string]int64, len(rows))
	for _, row := range rows {
		state[row.Slug] = row.State
		if row.ChannelID.Valid {
			bound[row.Slug] = row.ChannelID.Int64
		}
	}

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

	if view := r.URL.Query().Get("view"); view != "" {
		if c, ok := integrationBySlug(view); ok {
			dv := integrationDrawerView{
				ID: c.Slug, Name: c.Name, Mark: c.Mark, Category: c.Category,
				Description: c.Description, State: tileState(state[c.Slug]),
				Grants: c.Grants,
			}
			if id, ok := bound[c.Slug]; ok {
				dv.BoundChannel = strconv.FormatInt(id, 10)
			}
			data["IntDrawer"] = dv
		}
	}
	return nil
}

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

func channelDeliveryLabel(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.TrimSuffix(u.Host+u.Path, "/")
	}
	return raw
}

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
	// A drawer is read, not re-offered, so ?view is not stripped as dialog state (ADR-0130 §3).
	s.backToSection(w, r, "integrations")
}

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
	s.backToSection(w, r, "integrations")
}

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
		ok, err := s.channelExists(r.Context(), chID)
		if err != nil {
			s.serverError(w, "bind integration channel: list channels", err)
			return
		}
		if !ok {
			// A concurrent channel delete lands a real operator here, so it toasts, not 400s (ADR-0130 §1).
			s.toastRedirectBack(w, r, dest, "danger", "Channel not bound",
				"That channel no longer exists. Pick another.")
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
	s.redirectBack(w, r, dest)
}

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

func (s *server) testIntegration(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id := integrationFormID(r)
	integ, ok := integrationBySlug(id)
	if !ok {
		http.Error(w, "unknown integration", http.StatusBadRequest)
		return
	}
	dest := "/settings?tab=integrations&view=" + url.QueryEscape(id)

	binding, err := s.store.GetIntegrationChannel(r.Context(), id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.serverError(w, "test integration: read binding", err)
		return
	}
	if err != nil || !binding.Valid {
		s.toastRedirectBack(w, r, dest, "warn", "No delivery channel",
			"Connect a channel before sending a test.")
		return
	}

	ch, err := s.store.GetChannelForDelivery(r.Context(), binding.Int64)
	if err != nil {
		s.toastRedirectBack(w, r, dest, "danger", "Test message not sent",
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
		s.toastRedirectBack(w, r, dest, "danger", "Test message not sent",
			"Delivery through "+integ.Name+"'s channel failed — check the channel and try again.")
		return
	}
	s.toastRedirectBack(w, r, dest, "ok", "Test message sent",
		"Check "+integ.Name+" for the delivery.")
}

func integrationFormID(r *http.Request) string {
	if id := r.FormValue("id"); id != "" {
		return id
	}
	return r.FormValue("slug")
}

type channelTestSender interface {
	Send(ctx context.Context, targetURL string, body, secret []byte) (int, error)
}

// One transport, so a test send rides the same SSRF guard as the delivery runners.

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
