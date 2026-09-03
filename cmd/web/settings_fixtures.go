package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

// Dev-only Settings fixture (screen 21, package v3.13.0, WORK-ORDER-21-BATCH7). A
// VERGE_DEV build serves the design's curated fixtures.json → settings slice so
// /settings renders each section byte-for-byte for the pixel-parity harness (the 19
// golden states). It is the twin of the render-goldens settings case: both read the
// SAME fixtures.json slice and stamp the SAME "settings" holes, so the golden and the
// candidate are the same frozen tmpl fed the same data. It touches no table — the
// handler branches on s.devMode before any DB read (settingsPage) — so it mirrors the
// batch 3–6 fixtures-mode determinism pattern (signals/reports/etc.).
//
// Everything here is repo-owned harness glue, not a design-owned artifact; it only
// READS the frozen fixtures.json. A real deployment never reaches this path.

// The fixture shapes below mirror fixtures.json → settings.* exactly: snake_case json
// tags, PascalCase field names equal to the settings.tmpl declared holes, so a section
// slice is passed straight into the tmpl with no reshape.

type sfJob struct {
	ID          int64  `json:"id"`
	Href        string `json:"href"` // /runs/{run}?job={id} (DF-F3b), nullable
	Kind        string `json:"kind"`
	Vantage     string `json:"vantage"`
	State       string `json:"state"`
	Retrying    bool   `json:"retrying"`
	Superseded  bool   `json:"superseded"`
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	Batch       string `json:"batch"`
}

type sfActive struct {
	ID           int64   `json:"id"`   // dispatch id (DF-F3)
	Href         string  `json:"href"` // /runs/{dispatch} (DF-F3)
	ScanKind     string  `json:"scan_kind"`
	DispatchedAt string  `json:"dispatched_at"`
	Completed    int     `json:"completed"`
	Live         int     `json:"live"`
	Percent      int     `json:"percent"`
	Jobs         []sfJob `json:"jobs"`
	// Rollup is derived, not stored: the card's state-chip counts folded from Jobs at
	// fill time (#961), the same way fillScansSection folds them from the live rows. The
	// fixture keeps its jobs because the stop / terminate dialogs still count over them.
	Rollup jobRollup `json:"-"`
}

type sfHistory struct {
	Href         string `json:"href"` // /runs/{dispatch} (DF-F3), nullable
	ScanKind     string `json:"scan_kind"`
	DispatchedAt string `json:"dispatched_at"`
	Live         int    `json:"live"`
	Completed    int    `json:"completed"`
	Dead         int    `json:"dead"`
}

type sfColdScope struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	IsAddress bool   `json:"is_address"`
	OptedIn   bool   `json:"opted_in"`
}

type sfScans struct {
	Active  []sfActive  `json:"active"`
	History []sfHistory `json:"history"`
	// Truncated drives the history truncation callout (#962). The live handler derives it
	// from a LIMIT N+1 read; a fixture states it outright, since a fixture never carries
	// 51 rows just to render one line.
	Truncated   bool          `json:"truncated"`
	ColdEnabled bool          `json:"cold_enabled"`
	ColdScopes  []sfColdScope `json:"cold_scopes"`
}

type sfVantage struct {
	Name         string `json:"name"`
	Class        string `json:"class"`
	Resolver     string `json:"resolver"`
	Latency      string `json:"latency"`
	Availability string `json:"availability"`
	Unverified   bool   `json:"unverified"`
}

type sfProber struct {
	Endpoint           string `json:"endpoint"`
	Username           string `json:"username"`
	Availability       string `json:"availability"`
	HostKeyPinned      bool   `json:"host_key_pinned"`
	HostKeyFingerprint string `json:"host_key_fingerprint"`
	Platform           string `json:"platform"`
	KeySet             bool   `json:"key_set"`
	PublicKey          string `json:"public_key"`
	Egress             string `json:"egress"`
}

type sfVantages struct {
	Vantages []sfVantage `json:"vantages"`
	Probers  []sfProber  `json:"probers"`
}

type sfProvider struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Issuer    string `json:"issuer"`
	ClientID  string `json:"client_id"`
	HasSecret bool   `json:"has_secret"`
	Enabled   bool   `json:"enabled"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type sfBinding struct {
	ID           string `json:"id"`
	ProviderName string `json:"provider_name"`
	Account      string `json:"account"`
	DisplayName  string `json:"display_name"`
	LinkedAt     string `json:"linked_at"`
}

type sfSSO struct {
	Providers []sfProvider `json:"providers"`
	Bindings  []sfBinding  `json:"bindings"`
}

type sfMember struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Initials    string `json:"initials"`
	Role        string `json:"role"`
	TotpEnabled bool   `json:"totp_enabled"`
	At          string `json:"at"`
	IsSelf      bool   `json:"is_self"`
}

type sfTeam struct {
	Members    []sfMember `json:"members"`
	InviteLink string     `json:"invite_link_fixture"`
}

type sfSession struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Account   string `json:"account"`
	Role      string `json:"role"`
	Device    string `json:"device"`
	IP        string `json:"ip"`
	LastSeen  string `json:"last_seen"`
	Current   bool   `json:"current"`
}

type sfSource struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Kind  string   `json:"kind"`
	What  string   `json:"what"`
	On    bool     `json:"on"`
	Terms []string `json:"terms"`
}

type sfSources struct {
	Unencumbered     []sfSource `json:"unencumbered"`
	OperatorAccepted []sfSource `json:"operator_accepted"`
	Barred           []sfSource `json:"barred"`
}

type sfCounts struct {
	Sensitive int `json:"sensitive"`
	Frequency int `json:"frequency"`
	Union     int `json:"union"`
	TCP       int `json:"tcp"`
	UDP       int `json:"udp"`
}

type sfSensitive struct {
	Port      int    `json:"port"`
	Transport string `json:"transport"`
	Service   string `json:"service"`
}

type sfFrequency struct {
	Port          int    `json:"port"`
	AlsoSensitive bool   `json:"also_sensitive"`
	Edited        bool   `json:"edited"`
	EditAction    string `json:"edit_action"`
}

type sfAperture struct {
	UDPCount  int           `json:"udp_count"`
	Counts    sfCounts      `json:"counts"`
	Sensitive []sfSensitive `json:"sensitive"`
	Frequency []sfFrequency `json:"frequency"`
}

type sfInstanceVantage struct {
	Name    string `json:"name"`
	Latency string `json:"latency"`
	Avail   string `json:"avail"`
}

// Instance · data & release holes (v3.18.0, #391). The old sfUpdate callout is RETIRED
// (its content moved into the Release card); Backup / Release / Migrations / Restore
// mirror fixtures.json → settings.instance.* so the dev-mode golden render matches the
// frozen tmpl holes. Preflight / RestoreConfirm are pointers so a null fixture leaves
// them nil (the {{with}} branches collapse); Backup / Release / Migrations are struct
// values the fixture always carries.
type sfBackup struct {
	InProgress bool   `json:"in_progress"`
	Streamed   string `json:"streamed"`
	SizeHint   string `json:"size_hint"`
	Percent    int    `json:"percent"`
	LastAt     string `json:"last_at"`
	LastSize   string `json:"last_size"`
}

type sfLatest struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

type sfRelease struct {
	State        string   `json:"state"`
	CheckEnabled bool     `json:"check_enabled"`
	CheckedAt    string   `json:"checked_at"`
	Latest       sfLatest `json:"latest"`
	Steps        []string `json:"steps"`
}

type sfMigrations struct {
	Pending int `json:"pending"`
}

type sfPreflight struct {
	File     string `json:"file"`
	TakenAt  string `json:"taken_at"`
	Subjects string `json:"subjects"`
	Schema   string `json:"schema"`
}

type sfRestoreConfirm struct {
	File     string `json:"file"`
	TakenAt  string `json:"taken_at"`
	Subjects string `json:"subjects"`
}

type sfInstance struct {
	Version        string              `json:"version"`
	License        string              `json:"license"`
	Uptime         string              `json:"uptime"`
	QueueDepth     int                 `json:"queue_depth"`
	DiskPct        int                 `json:"disk_pct"`
	DiskDetail     string              `json:"disk_detail"`
	PgLabel        string              `json:"pg_label"`
	PgDetail       string              `json:"pg_detail"`
	Vantages       []sfInstanceVantage `json:"vantages"`
	Backup         sfBackup            `json:"backup"`
	RestoreError   string              `json:"restore_error"`
	Preflight      *sfPreflight        `json:"preflight"`
	RestoreConfirm *sfRestoreConfirm   `json:"restore_confirm"`
	Migrations     sfMigrations        `json:"migrations"`
	Release        sfRelease           `json:"release"`
}

// sfAPI mirrors fixtures.json → settings.api: the read-only /api/v1 opt-in state and,
// when enabled, the dated act of the current state (By/At). Disabled in the fixture, so
// By/At are empty and the tmpl renders only the disabled badge + note.
type sfAPI struct {
	Enabled bool   `json:"enabled"`
	By      string `json:"by"`
	At      string `json:"at"`
}

type sfClassOption struct {
	Name    string `json:"name"`
	Checked bool   `json:"checked"`
}

type sfChannel struct {
	ID          string          `json:"id"`
	URL         string          `json:"url"`
	Classes     []string        `json:"classes"`
	ClassStates []sfClassOption `json:"class_states"`
	HasSecret   bool            `json:"has_secret"`
	Enabled     bool            `json:"enabled"`
	By          string          `json:"by"`
	At          string          `json:"at"`
}

type sfChannels struct {
	ClassOptions []sfClassOption `json:"class_options"`
	Channels     []sfChannel     `json:"channels"`
}

type sfTile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Mark        string `json:"mark"`
	Category    string `json:"category"`
	State       string `json:"state"`
	Description string `json:"description"`
}

type sfGrant struct {
	Scope  string `json:"scope"`
	Detail string `json:"detail"`
	Write  bool   `json:"write"`
}

type sfDrawer struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Mark         string    `json:"mark"`
	Category     string    `json:"category"`
	State        string    `json:"state"`
	Description  string    `json:"description"`
	Attention    string    `json:"attention"`
	Grants       []sfGrant `json:"grants"`
	Installed    string    `json:"installed"`
	LastDelivery string    `json:"last_delivery"`
	Classes      string    `json:"classes"`
	// BoundChannel (#39b) is the id of the delivery Channel this integration is bound
	// to — matched against an IntChannels option Value so the drawer's select renders it
	// selected; "" is unbound (Not connected), which gates "Send test" off.
	BoundChannel string `json:"bound_channel"`
}

// sfIntChannel is one option of the drawer's "Delivery channel" select (#39b): the
// fixtures' integrations.channels slice, stamped verbatim into .IntChannels[{Value,
// Label,Hint}]. The fixture is the exact demo corpus this render reproduces.
type sfIntChannel struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Hint  string `json:"hint"`
}

type sfIntegrations struct {
	Cats          []string       `json:"cats"`
	Cat           string         `json:"cat"`
	Q             string         `json:"q"`
	Tiles         []sfTile       `json:"tiles"`
	Channels      []sfIntChannel `json:"channels"`
	Drawer        sfDrawer       `json:"drawer_fixture"`
	DrawerUnbound sfDrawer       `json:"drawer_unbound_fixture"`
}

type sfCensus struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
	Href string `json:"href"`
}

type sfDelivery struct {
	State       string `json:"state"`
	ChannelHost string `json:"channel_host"`
	Failed      bool   `json:"failed"`
	LastError   string `json:"last_error"`
}

type sfMessage struct {
	ID         string       `json:"id"`
	Read       bool         `json:"read"`
	Cause      string       `json:"cause"`
	Class      string       `json:"class"`
	Instant    string       `json:"instant"`
	Headline   string       `json:"headline"`
	Href       string       `json:"href"`
	LinkText   string       `json:"link_text"`
	Census     []sfCensus   `json:"census"`
	Deliveries []sfDelivery `json:"deliveries"`
}

type sfOutcome struct {
	ChannelHost string `json:"channel_host"`
	Class       string `json:"class"`
	Failed      bool   `json:"failed"`
	State       string `json:"state"`
	When        string `json:"when"`
}

type sfRetention struct {
	ObservationCurrencyDays int    `json:"observation_currency_days"`
	DispatchCadenceMultiple int    `json:"dispatch_cadence_multiple"`
	UpdatedAt               string `json:"updated_at"`
	UpdatedBy               string `json:"updated_by"`
}

type sfDeliverySection struct {
	Deliveries []sfOutcome `json:"deliveries"`
	Retention  sfRetention `json:"retention"`
}

type settingsFixture struct {
	DefaultTab   string            `json:"default_tab"`
	Scans        sfScans           `json:"scans"`
	Vantages     sfVantages        `json:"vantages"`
	SSO          sfSSO             `json:"sso"`
	Team         sfTeam            `json:"team"`
	Sessions     []sfSession       `json:"sessions"`
	AuditRows    json.RawMessage   `json:"audit_rows"`
	Sources      sfSources         `json:"sources"`
	Aperture     sfAperture        `json:"aperture"`
	Instance     sfInstance        `json:"instance"`
	Channels     sfChannels        `json:"channels"`
	Integrations sfIntegrations    `json:"integrations"`
	Messages     []sfMessage       `json:"messages"`
	Delivery     sfDeliverySection `json:"delivery"`
	API          sfAPI             `json:"api"`
}

func loadSettingsFixture() (settingsFixture, error) {
	raw, err := fs.ReadFile(designfs.FS, "fixtures/fixtures.json")
	if err != nil {
		return settingsFixture{}, err
	}
	var ff struct {
		Settings settingsFixture `json:"settings"`
	}
	if err := json.Unmarshal(raw, &ff); err != nil {
		return settingsFixture{}, err
	}
	return ff.Settings, nil
}

// settingsFixtureData stamps the "settings" holes from the fixtures.json settings
// slice, honouring the query the harness navigates for each of the 18 chrome-hosted
// states (the 19th, forbidden, is the viewer's requireSettingsAdmin refusal, which
// renders the error-page settings-forbidden and never reaches here). It mirrors the
// render-goldens settings case one-for-one so the golden and candidate agree.
func (s *server) settingsFixtureData(acct db.Account, r *http.Request) map[string]any {
	fx, err := loadSettingsFixture()
	if err != nil {
		log.Printf("web: settings fixture: %v", err)
	}
	q := r.URL.Query()
	tab := validTab(q.Get("tab"))

	data := map[string]any{
		"Title": "Settings", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "settings", "Tab": tab, "DesignTokens": true,
	}

	switch tab {
	case "scans":
		fillFixtureRollups(fx.Scans.Active)
		data["Active"] = fx.Scans.Active
		data["History"] = fx.Scans.History
		data["Truncated"] = fx.Scans.Truncated
		data["HistoryLimit"] = scansHistoryLimit
		data["ColdEnabled"] = fx.Scans.ColdEnabled
		data["ColdScopes"] = fx.Scans.ColdScopes
		data["ColdError"] = ""
		// The stop / terminate confirm dialogs (DF-F4, states scans-stop-confirm /
		// scans-terminate-confirm at id 1409, #35). The harness navigates ?stop=/?terminate=;
		// the target is built from the matching active dispatch. Pending is the ready jobs
		// a stop cancels and Running the running jobs it lets finish — the rollup already
		// folded both, so the dialog reads them off it rather than folding a second time.
		if id := q.Get("stop"); id != "" {
			if a := findActiveDispatch(fx.Scans.Active, id); a != nil {
				data["StopTarget"] = map[string]any{
					"ID": a.ID, "ScanKind": a.ScanKind,
					"Pending": a.Rollup.Ready, "Running": a.Rollup.Running,
				}
			}
		}
		if id := q.Get("terminate"); id != "" {
			if a := findActiveDispatch(fx.Scans.Active, id); a != nil {
				data["TerminateTarget"] = map[string]any{
					"ID": a.ID, "ScanKind": a.ScanKind, "Running": a.Rollup.Running,
				}
			}
		}
	case "vantages":
		data["Vantages"] = fx.Vantages.Vantages
		data["Probers"] = fx.Vantages.Probers
		data["ProberError"] = ""
		data["ProberHost"] = ""
		data["ProberPort"] = ""
		data["ProberUser"] = ""
	case "sso":
		data["SSOProviders"] = fx.SSO.Providers
		data["SSOBindings"] = fx.SSO.Bindings
		data["SSOError"] = ""
		data["SSOName"] = ""
		data["SSOSlug"] = ""
		data["SSOIssuer"] = ""
		data["SSOClientID"] = ""
	case "team":
		data["Members"] = fx.Team.Members
		data["TeamError"] = ""
		data["RoleError"] = ""
		data["RemoveError"] = ""
		data["InviteLink"] = ""
		data["InviteRole"] = ""
		data["InviteOpen"] = q.Get("invite") != ""
		if id := q.Get("remove"); id != "" {
			if m := findMember(fx.Team.Members, id); m != nil {
				data["RemoveTarget"] = m
			}
		}
		if id := q.Get("role"); id != "" {
			if m := findMember(fx.Team.Members, id); m != nil {
				data["RoleTarget"] = m
			}
		}
		if id := q.Get("reenroll"); id != "" {
			if m := findMember(fx.Team.Members, id); m != nil {
				data["ReenrollTarget"] = m
			}
		}
	case "sessions":
		data["Sessions"] = fx.Sessions
		data["RevokeAccountError"] = ""
		if id := q.Get("revoke"); id != "" {
			for i := range fx.Sessions {
				if fx.Sessions[i].ID == id {
					data["RevokeSessionTarget"] = fx.Sessions[i]
					break
				}
			}
		}
		if id := q.Get("revoke-account"); id != "" {
			for i := range fx.Sessions {
				if fx.Sessions[i].AccountID == id {
					data["RevokeAccountTarget"] = map[string]any{
						"AccountID": id, "Username": fx.Sessions[i].Account,
					}
					break
				}
			}
		}
	case "audit":
		data["AuditRows"] = nil
	case "api":
		data["API"] = fx.API
	case "sources":
		data["Unencumbered"] = fx.Sources.Unencumbered
		data["OperatorAccepted"] = fx.Sources.OperatorAccepted
		data["Barred"] = fx.Sources.Barred
		data["SourceError"] = ""
		if id := q.Get("consent"); id != "" {
			for i := range fx.Sources.OperatorAccepted {
				src := fx.Sources.OperatorAccepted[i]
				if src.ID == id {
					data["Consent"] = map[string]any{
						"ID": src.ID, "Name": src.Name, "Terms": src.Terms,
					}
					break
				}
			}
		}
	case "aperture":
		data["UDPCount"] = fx.Aperture.UDPCount
		data["Counts"] = fx.Aperture.Counts
		data["Sensitive"] = fx.Aperture.Sensitive
		data["Frequency"] = fx.Aperture.Frequency
		data["VCError"] = ""
		data["VCPort"] = ""
	case "instance":
		data["Instance"] = fx.Instance
	case "channels":
		data["Channels"] = fx.Channels.Channels
		data["ClassOptions"] = fx.Channels.ClassOptions
		data["ChanError"] = ""
		data["ChanURL"] = ""
	case "integrations":
		data["IntCats"] = fx.Integrations.Cats
		data["IntCat"] = fx.Integrations.Cat
		data["IntQ"] = fx.Integrations.Q
		data["Integrations"] = fx.Integrations.Tiles
		// The drawer's "Delivery channel" select options (#39b) — the same slice for
		// every drawer; only used inside {{with .IntDrawer}}, so the base tab render is
		// unchanged.
		data["IntChannels"] = fx.Integrations.Channels
		// The spec drawer (?view=<id>): pagerduty is the bound fixture, slack the freshly-
		// installed unbound fixture. Each carries its own bound_channel, so the select
		// renders the right option selected (or "Not connected") and gates "Send test".
		if id := q.Get("view"); id != "" {
			switch id {
			case fx.Integrations.Drawer.ID:
				data["IntDrawer"] = fx.Integrations.Drawer
			case fx.Integrations.DrawerUnbound.ID:
				data["IntDrawer"] = fx.Integrations.DrawerUnbound
			}
		}
	case "messages":
		data["Messages"] = fx.Messages
	case "delivery":
		data["Deliveries"] = fx.Delivery.Deliveries
		data["Retention"] = fx.Delivery.Retention
		data["RetError"] = ""
		data["RetObs"] = ""
		data["RetDispatch"] = ""
	}
	return data
}

// findActiveDispatch returns the fixture active dispatch whose id matches the raw query
// value (the ?stop=/?terminate= target, id 1409 — #35), or nil.
func findActiveDispatch(active []sfActive, raw string) *sfActive {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil
	}
	for i := range active {
		if active[i].ID == id {
			return &active[i]
		}
	}
	return nil
}

// fillFixtureRollups folds each active fixture dispatch's jobs into the Running-now
// card's state-chip counts (#961), the same fold fillScansSection runs over the live
// rows. The stop / terminate dialogs read their Pending and Running off it too.
// loadSettingsFixture decodes a fresh copy per request, so this writes to no shared state.
func fillFixtureRollups(active []sfActive) {
	for i := range active {
		active[i].Rollup = toJobRollup(active[i].Jobs, func(j sfJob) string { return j.State })
	}
}

func findMember(members []sfMember, id string) *sfMember {
	for i := range members {
		if members[i].ID == id {
			return &members[i]
		}
	}
	return nil
}
