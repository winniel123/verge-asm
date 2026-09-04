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

// The names stamp into settings.tmpl holes verbatim, so a rename can empty a section silently.

type sfJob struct {
	ID          int64  `json:"id"`
	Href        string `json:"href"`
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
	ID           int64     `json:"id"`
	Href         string    `json:"href"`
	ScanKind     string    `json:"scan_kind"`
	DispatchedAt string    `json:"dispatched_at"`
	Completed    int       `json:"completed"`
	Live         int       `json:"live"`
	Percent      int       `json:"percent"`
	Jobs         []sfJob   `json:"jobs"`
	Rollup       jobRollup `json:"-"`
}

type sfHistory struct {
	Href         string `json:"href"`
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

	// A fixture states this outright rather than carry 51 rows the live LIMIT N+1 read needs (#962).

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

// A pointer field lets a null fixture collapse its {{with}} branch; a struct value always renders.

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
	BoundChannel string    `json:"bound_channel"`
}

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

func (s *server) settingsFixtureData(acct db.Account, r *http.Request) map[string]any {
	fx, err := loadSettingsFixture()
	if err != nil {
		log.Printf("web: settings fixture: %v", err)
	}
	q := r.URL.Query()
	tab := validTab(q.Get("tab"))

	data := map[string]any{
		"Title": "Settings", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "settings", "Tab": tab,
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
		data["IntChannels"] = fx.Integrations.Channels
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

func fillFixtureRollups(active []sfActive) {
	// The fixture is decoded fresh per request, so this in-place fold shares no state.
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
