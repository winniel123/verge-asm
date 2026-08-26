package main

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

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
// slice is passed straight into the tmpl with no reshape. TestSettingsFixtureMatchesPackage
// (settings_fixtures_test.go) folds representative values back through fixtures.json.

type sfJob struct {
	ID          int64  `json:"id"`
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
	ScanKind     string  `json:"scan_kind"`
	DispatchedAt string  `json:"dispatched_at"`
	Completed    int     `json:"completed"`
	Live         int     `json:"live"`
	Percent      int     `json:"percent"`
	Jobs         []sfJob `json:"jobs"`
}

type sfHistory struct {
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
	Active      []sfActive    `json:"active"`
	History     []sfHistory   `json:"history"`
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

type sfUpdate struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

type sfInstance struct {
	Update     *sfUpdate           `json:"update"`
	Version    string              `json:"version"`
	License    string              `json:"license"`
	Uptime     string              `json:"uptime"`
	QueueDepth int                 `json:"queue_depth"`
	DiskPct    int                 `json:"disk_pct"`
	DiskDetail string              `json:"disk_detail"`
	PgLabel    string              `json:"pg_label"`
	PgDetail   string              `json:"pg_detail"`
	Vantages   []sfInstanceVantage `json:"vantages"`
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
}

type sfIntegrations struct {
	Cats   []string `json:"cats"`
	Cat    string   `json:"cat"`
	Q      string   `json:"q"`
	Tiles  []sfTile `json:"tiles"`
	Drawer sfDrawer `json:"drawer_fixture"`
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

// settingsFixture is the whole fixtures.json → settings slice.
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
}

// loadSettingsFixture reads and decodes the fixtures.json → settings slice.
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
		data["Active"] = fx.Scans.Active
		data["History"] = fx.Scans.History
		data["ColdEnabled"] = fx.Scans.ColdEnabled
		data["ColdScopes"] = fx.Scans.ColdScopes
		data["ColdError"] = ""
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
		if id := q.Get("view"); id != "" && id == fx.Integrations.Drawer.ID {
			data["IntDrawer"] = fx.Integrations.Drawer
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

// findMember returns the fixture member with the given id, or nil.
func findMember(members []sfMember, id string) *sfMember {
	for i := range members {
		if members[i].ID == id {
			return &members[i]
		}
	}
	return nil
}
