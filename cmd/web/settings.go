package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/seed"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// Every mutation this screen hosts is an authenticated admin act (docs/spec/v1-spec.md §4.3).

// The secret is write-only and never rendered again (CONTEXT.md "Channel").

type channelView struct {
	ID          int64
	URL         string
	Drift       bool
	Coverage    bool
	Clock       bool
	Classes     []string
	ClassStates []classState
	Enabled     bool
	HasSecret   bool
	By          string
	At          string
}

type classState struct {
	Name    string
	Checked bool
}

var channelClasses = []string{"drift", "coverage", "clock"}

type accountRow struct {
	ID          int64
	Username    string
	Initials    string
	Role        string
	TotpEnabled bool
	At          string
	IsSelf      bool
}

// The current row shows no revoke control, so an admin never ends their own session here.

type sessionRow struct {
	ID        int64
	AccountID int64
	Account   string
	Role      string
	Device    string
	IP        string
	LastSeen  string
	Current   bool
}

type retentionView struct {
	ObservationCurrencyDays int64
	DispatchCadenceMultiple int64
	TranscriptCurrencyDays  int64
	UpdatedBy               string
	UpdatedAt               string
}

// Priced at policy time so a declaration stays a flat confirm: a readout, never a gate (ADR-0127).

type addressCapView struct {
	Cap            int64
	CapLabel       string
	LargestScopeV4 string
	LargestScopeV6 string
	DiskProjection string
	SweepLoad      []capSweepLine
	UpdatedBy      string
	UpdatedAt      string
}

type capSweepLine struct {
	Scan      string
	Cadence   string
	Probes    string
	Effective string
	Outpaces  bool
}

type vantageRow struct {
	Name         string
	Class        string
	Availability string
	Resolver     string
	Endpoint     string
	Latency      string
	Unverified   bool
	Avail        string
}

type settingsForms struct {
	section string
	tab     string
	notice  string

	flashTab string

	teamError   string
	roleError   string
	inviteRole  string
	inviteLink  string
	inviteOpen  bool
	removeID    int64
	removeError string

	chanError    string
	chanURL      string
	chanDrift    bool
	chanCoverage bool
	chanClock    bool

	ssoError    string
	ssoSlug     string
	ssoName     string
	ssoIssuer   string
	ssoClientID string

	retError      string
	retObs        string
	retDispatch   string
	retTranscript string

	capError string
	capValue string

	vcError string
	vcPort  string

	sourceError string

	coldError   string
	proberError string
	proberHost  string
	proberPort  string
	proberUser  string

	revokeAccountID    int64
	revokeAccountError string

	restoreError   string
	preflight      *restorePreflightView
	restoreConfirm *restoreConfirmView
}

var settingsTabs = []string{
	"scans", "vantages",
	"sso", "team", "audit", "api", "sessions",
	"sources", "aperture",
	"instance",
	"channels", "integrations", "messages", "delivery",
}

func validTab(t string) string {
	// A bookmarked pre-V3 ?tab=access link must land on Team rather than 404.
	if t == "access" {
		return "team"
	}
	for _, x := range settingsTabs {
		if x == t {
			return t
		}
	}
	return "scans"
}

func tabForSection(section string) string {
	switch section {
	case "api":
		return "api"
	case "sso":
		return "sso"
	case "team":
		return "team"
	case "channels":
		return "channels"
	case "retention":
		return "delivery"
	case "addresscap":
		return "scans"
	case "vergecore":
		return "aperture"
	case "sources":
		return "sources"
	case "integrations":
		return "integrations"
	case "sessions":
		return "sessions"
	case "vantages":
		return "vantages"
	case "instance":
		return "instance"
	case "scans", "cold":
		return "scans"
	default:
		return "scans"
	}
}

// A dropped dialog param is modal state, not the list ADR-0130 §3 preserves.

func dialogParams(tab string) []string {
	// Keyed on the tab: three sections share Scans, so a section key keeps the wrong confirm open.
	switch tab {
	case "team":
		return []string{"role", "reenroll", "remove", "invite"}
	case "sources":
		return []string{"consent"}
	case "sessions":
		return []string{"revoke", "revoke-account"}
	case "scans":
		return []string{"stop", "terminate"}
	case "instance":
		return []string{"restore-confirm"}
	}
	return nil
}

func (s *server) backToSection(w http.ResponseWriter, r *http.Request, section string) {
	// A refusal lands exactly where a success does, so the shell restores the offset (ADR-0130 §1).
	tab := tabForSection(section)
	dest := s.resolveBack(r, "/settings?tab="+tab)
	http.Redirect(w, r, stripDestParams(dest, dialogParams(tab)...), http.StatusSeeOther)
}

func (s *server) takeSettingsFlash(r *http.Request, tab string) settingsForms {
	// /settings and /scans are separate landings, so a stash claimed by the wrong one is lost.
	f, _ := takeFormFlashIf[settingsForms](s, r, func(v settingsForms) bool {
		return v.flashTab == tab
	})
	f.tab = tab
	return f
}

func (s *server) failSettings(w http.ResponseWriter, r *http.Request, f settingsForms) {
	s.flashSettings(w, r, f)
}

func (s *server) toastBackToSection(w http.ResponseWriter, r *http.Request, accountID int64, section, tone, title, description string) {
	// A toast spelled on the URL fires again on every meta-refresh the in-flight Scans page runs.
	s.flash.set(accountID, toastVM{Tone: tone, Title: title, Description: description})
	s.backToSection(w, r, section)
}

func (s *server) flashSettings(w http.ResponseWriter, r *http.Request, f settingsForms) {
	f.flashTab = tabForSection(f.section)
	stashFormFlash(s, r, f)
	s.backToSection(w, r, f.section)
}

func (s *server) settingsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "settings", s.settingsFixtureData(acct, r))
		return
	}
	q := r.URL.Query()
	tab := validTab(q.Get("tab"))
	// The bounce must precede the flash take, or the redirect swallows a stash it never renders.
	if tab == "integrations" && !integrationsEnabled {
		http.Redirect(w, r, "/settings?tab=scans", http.StatusSeeOther)
		return
	}
	forms := s.takeSettingsFlash(r, tab)
	if forms.tab == "instance" {
		if stg := s.stagedRestore(acct.ID); stg != nil {
			if q.Get("restore-confirm") == "1" {
				forms.restoreConfirm = &restoreConfirmView{
					File: stg.file, TakenAt: stg.takenAt, Subjects: stg.subjects,
				}
			} else {
				forms.preflight = &restorePreflightView{
					File: stg.file, TakenAt: stg.takenAt, Subjects: stg.subjects, Schema: stg.schema,
				}
			}
		}
	}
	s.renderSettings(w, r, acct, forms)
}

// Bounding the life is what makes a leaked join link inert once it goes stale.

const inviteTTL = 7 * 24 * time.Hour

func (s *server) inviteAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	role := r.FormValue("role")
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{
			section: "team", teamError: msg, inviteOpen: true, inviteRole: role,
		})
	}
	if role != roleAdmin && role != roleViewer {
		fail("Role must be admin or viewer.")
		return
	}
	plaintext, hash, err := newOpaqueToken()
	if err != nil {
		s.serverError(w, "mint invite token", err)
		return
	}
	if _, err := s.store.CreateInvite(r.Context(), db.CreateInviteParams{
		TokenHash: hash, Role: role,
		InvitedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
		ExpiresAt: pgtype.Timestamptz{Time: s.now().Add(inviteTTL), Valid: true},
	}); err != nil {
		s.serverError(w, "create invite", err)
		return
	}
	link := s.inviteLink(r, plaintext)
	log.Printf("web: invite minted at role %q; accept it at %s (expires in %s)", role, link, inviteTTL) // #nosec G706 (role is enum-validated admin|viewer; link is server-constructed)
	stashFormFlash(s, r, settingsForms{flashTab: "team", inviteOpen: true, inviteLink: link})
	s.backToSection(w, r, "team")
}

func (s *server) inviteLink(r *http.Request, token string) string {
	// A TLS proxy leaves web on plain HTTP, so the cookie flag is the tell (docs/guides/running.md).
	scheme := "http"
	if r.TLS != nil || s.secureCookies {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/invite?token=" + token
}

func (s *server) setAccountRole(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "team", roleError: msg})
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		fail("That account could not be found.")
		return
	}
	role := r.FormValue("role")
	if role != roleAdmin && role != roleViewer {
		fail("Role must be admin or viewer.")
		return
	}
	target, err := s.store.GetAccountByID(r.Context(), id)
	if err != nil {
		fail("That account could not be found.")
		return
	}
	if target.Role == roleAdmin && role == roleViewer {
		n, err := s.store.CountAdmins(r.Context())
		if err != nil {
			s.serverError(w, "count admins", err)
			return
		}
		if n <= 1 {
			fail("You cannot demote the last admin — promote another account first.")
			return
		}
	}
	if err := s.store.UpdateAccountRole(r.Context(), db.UpdateAccountRoleParams{ID: id, Role: role}); err != nil {
		s.serverError(w, "update account role", err)
		return
	}
	s.backToSection(w, r, "team")
}

func (s *server) reenrollAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "team", teamError: "That account could not be found."})
		return
	}
	if err := s.store.ResetAccountTOTP(r.Context(), id); err != nil {
		s.serverError(w, "reset account totp", err)
		return
	}
	s.backToSection(w, r, "team")
}

func (s *server) removeAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "team", teamError: "That account could not be found."})
		return
	}
	// The URL re-opens this dialog itself; removeID covers the fallback where no return survived.
	reopen := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "team", removeID: id, removeError: msg})
	}
	if id == acct.ID {
		s.failSettings(w, r, settingsForms{section: "team", teamError: "You cannot remove your own account."})
		return
	}
	target, err := s.store.GetAccountByID(r.Context(), id)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "team", teamError: "That account could not be found."})
		return
	}
	if strings.TrimSpace(r.FormValue("confirm_name")) != target.Username {
		reopen("That did not match. Type " + target.Username + " exactly to remove.")
		return
	}
	if target.Role == roleAdmin {
		n, err := s.store.CountAdmins(r.Context())
		if err != nil {
			s.serverError(w, "count admins", err)
			return
		}
		if n <= 1 {
			reopen("You cannot remove the last admin — promote another account first.")
			return
		}
	}
	// Attributed acts pin their author with created_by (docs/guides/accounts.md).
	if err := s.store.DeleteAccount(r.Context(), id); err != nil {
		if isForeignKeyViolation(err) {
			reopen(target.Username + " has declared scopes, channels, or other attributed acts and cannot be removed — reassign or keep the account.")
			return
		}
		s.serverError(w, "delete account", err)
		return
	}
	s.backToSection(w, r, "team")
}

func (s *server) createChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rawURL := strings.TrimSpace(r.FormValue("url"))
	drift, coverage, clock := classesFromForm(r)
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{
			section: "channels", chanError: msg, chanURL: rawURL,
			chanDrift: drift, chanCoverage: coverage, chanClock: clock,
		})
	}

	normURL, msg := validateChannelURL(rawURL)
	if msg != "" {
		fail(msg)
		return
	}
	if !drift && !coverage && !clock {
		fail("Choose at least one routing class.")
		return
	}
	if _, err := s.store.CreateChannel(r.Context(), db.CreateChannelParams{
		Url: normURL, Secret: optionalSecret(r.FormValue("secret")),
		RouteDrift: drift, RouteCoverage: coverage, RouteClock: clock,
		Enabled: true, CreatedBy: acct.ID,
	}); err != nil {
		s.serverError(w, "create channel", err)
		return
	}
	s.backToSection(w, r, "channels")
}

func (s *server) updateChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{section: "channels", chanError: msg})
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		fail("That channel could not be found.")
		return
	}
	rawURL := strings.TrimSpace(r.FormValue("url"))
	drift, coverage, clock := classesFromForm(r)
	normURL, msg := validateChannelURL(rawURL)
	if msg != "" {
		fail(msg)
		return
	}
	if !drift && !coverage && !clock {
		fail("Choose at least one routing class.")
		return
	}
	if err := s.store.UpdateChannel(r.Context(), db.UpdateChannelParams{
		ID: id, Url: normURL, RouteDrift: drift, RouteCoverage: coverage,
		RouteClock: clock, Enabled: r.FormValue("enabled") != "",
	}); err != nil {
		s.serverError(w, "update channel", err)
		return
	}
	switch {
	case r.FormValue("clear_secret") != "":
		if err := s.store.SetChannelSecret(r.Context(), db.SetChannelSecretParams{ID: id}); err != nil {
			s.serverError(w, "clear channel secret", err)
			return
		}
	case strings.TrimSpace(r.FormValue("secret")) != "":
		if err := s.store.SetChannelSecret(r.Context(), db.SetChannelSecretParams{
			ID: id, Secret: pgtype.Text{String: r.FormValue("secret"), Valid: true},
		}); err != nil {
			s.serverError(w, "set channel secret", err)
			return
		}
	}
	s.backToSection(w, r, "channels")
}

func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.failSettings(w, r, settingsForms{section: "channels", chanError: "That channel could not be found."})
		return
	}
	if err := s.store.DeleteChannel(r.Context(), id); err != nil {
		s.serverError(w, "delete channel", err)
		return
	}
	s.backToSection(w, r, "channels")
}

// The floor is derived from the tightest bound in force, never an operator choice (ADR-0094).

func (s *server) updateRetention(w http.ResponseWriter, r *http.Request, acct db.Account) {
	obsRaw := strings.TrimSpace(r.FormValue("observation_currency_days"))
	dispRaw := strings.TrimSpace(r.FormValue("dispatch_cadence_multiple"))
	transRaw := strings.TrimSpace(r.FormValue("transcript_currency_days"))
	fail := func(msg string) {
		s.failSettings(w, r, settingsForms{
			section: "retention", retError: msg,
			retObs: obsRaw, retDispatch: dispRaw, retTranscript: transRaw,
		})
	}

	obs, err := strconv.ParseInt(obsRaw, 10, 64)
	if err != nil || obs < 0 {
		fail("Observation-currency floor must be a whole number of days, zero or more.")
		return
	}
	disp, err := strconv.ParseInt(dispRaw, 10, 64)
	if err != nil || disp < 0 {
		fail("Dispatch floor must be a whole number of cadences, zero or more.")
		return
	}
	// A positive value is floored up by the retirer, so none is refused here (raw-job-output.md §4).
	trans, err := strconv.ParseInt(transRaw, 10, 64)
	if err != nil || trans < 0 {
		fail("Transcript retention must be a whole number of days, zero or more.")
		return
	}
	tightest, err := s.store.TightestEnabledScanCadenceSeconds(r.Context())
	if err != nil {
		s.serverError(w, "tightest scan cadence", err)
		return
	}
	if retention.BelowObservationFloor(obs, tightest) {
		floorDays, _ := retention.ObservationFloorDays(tightest)
		fail(fmt.Sprintf("Observation currency must be at least %d days — the tightest observation bound in force — or 0 to leave it unbounded.", floorDays))
		return
	}
	if retention.BelowFloor(disp) {
		fail(fmt.Sprintf("Dispatch retention must be at least %d cadences of the slowest enabled Scan, or 0 to leave it unbounded.", retention.FloorCadences))
		return
	}
	if err := s.store.UpdateRetentionSettings(r.Context(), db.UpdateRetentionSettingsParams{
		ObservationCurrencyDays: obs, DispatchCadenceMultiple: disp,
		TranscriptCurrencyDays: trans,
		UpdatedBy:              pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update retention", err)
		return
	}
	s.backToSection(w, r, "retention")
}

func (s *server) updateAddressCap(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := strings.TrimSpace(r.FormValue("address_cap"))
	// A ceiling here is refused; a large scope is priced at policy time, not gated (ADR-0127).
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		s.failSettings(w, r, settingsForms{
			section:  "addresscap",
			capError: "The address-scope cap must be a whole number of addresses, one or more.",
			capValue: raw,
		})
		return
	}
	if err := s.store.SetSeedAddressCap(r.Context(), db.SetSeedAddressCapParams{
		SeedAddressCap:          n,
		SeedAddressCapUpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update address cap", err)
		return
	}
	s.backToSection(w, r, "addresscap")
}

func (s *server) renderSettings(w http.ResponseWriter, r *http.Request, acct db.Account, f settingsForms) {
	active := f.tab
	if active == "" {
		active = tabForSection(f.section)
	}

	data := map[string]any{
		"Title": "Settings", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "settings", "Tab": active,
	}
	if f.notice != "" {
		data["Notice"] = f.notice
	}

	var err error
	switch active {
	case "scans":
		err = s.fillScansSection(r, acct, f, data)
	case "vantages":
		err = s.fillVantagesSection(r, f, data)
	case "sso":
		err = s.fillSSOSection(r, f, data)
	case "team":
		err = s.fillTeamSection(r, acct, f, data)
	case "audit":
		err = s.fillAuditSection(r, data)
	case "api":
		err = s.fillAPISection(r, data)
	case "sessions":
		err = s.fillSessionsSection(r, f, data)
	case "sources":
		err = s.fillSourcesSection(r, f, data)
	case "aperture":
		err = s.fillApertureSection(r, f, data)
	case "instance":
		err = s.fillInstanceSection(r, f, data)
	case "channels":
		err = s.fillChannelsSection(r, f, data)
	case "integrations":
		err = s.fillIntegrationsSection(r, data)
	case "messages":
		err = s.fillMessagesSection(r, acct, data)
	case "delivery":
		err = s.fillDeliverySection(r, f, data)
	}
	if err != nil {
		s.serverError(w, "settings section "+active, err)
		return
	}

	s.renderStatus(w, r, http.StatusOK, "settings", data)
}

func (s *server) fillVantagesSection(r *http.Request, f settingsForms, data map[string]any) error {
	rows, err := s.store.ListVantages(r.Context())
	if err != nil {
		return err
	}
	data["ProberError"] = f.proberError
	data["ProberHost"] = f.proberHost
	data["ProberPort"] = f.proberPort
	data["ProberUser"] = f.proberUser
	out := make([]vantageRow, 0, len(rows))
	for _, v := range rows {
		vr := vantageRow{
			Name: v.Name, Class: v.Class, Availability: v.Availability.String,
			Resolver: v.Resolver, Endpoint: endpointString(v.Host.String, v.Port.Int32),
			Latency: vantageLatencyLabel(v.LatencyMs),
		}
		if vr.Availability == "" {
			vr.Availability = "pending"
		}
		vr.Unverified = vr.Availability == "unverified"
		out = append(out, vr)
	}
	data["Vantages"] = out
	data["Probers"] = toProberViews(rows)
	return nil
}

func (s *server) fillChannelsSection(r *http.Request, f settingsForms, data map[string]any) error {
	channels, err := s.store.ListChannels(r.Context())
	if err != nil {
		return err
	}
	chDrift, chCoverage, chClock := true, true, true
	if f.section == "channels" {
		chDrift, chCoverage, chClock = f.chanDrift, f.chanCoverage, f.chanClock
	}
	data["Channels"] = toChannelViews(channels)
	data["ChanError"] = f.chanError
	data["ChanURL"] = f.chanURL
	data["ChanDrift"] = chDrift
	data["ChanCoverage"] = chCoverage
	data["ChanClock"] = chClock
	defaults := map[string]bool{"drift": chDrift, "coverage": chCoverage, "clock": chClock}
	opts := make([]classState, 0, len(channelClasses))
	for _, name := range channelClasses {
		opts = append(opts, classState{Name: name, Checked: defaults[name]})
	}
	data["ClassOptions"] = opts
	return nil
}

func initialsFromUsername(username string) string {
	local := username
	if i := strings.IndexByte(username, '@'); i >= 0 {
		local = username[:i]
	}
	letters := make([]rune, 0, 2)
	for _, r := range local {
		letters = append(letters, r)
		if len(letters) == 2 {
			break
		}
	}
	return strings.ToUpper(string(letters))
}

func (s *server) fillDeliverySection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()

	var deliveries []deliveryView
	if outcomes, derr := s.store.ListDeliveryOutcomes(ctx); derr == nil {
		for _, o := range outcomes {
			deliveries = append(deliveries, toDeliveryView(o))
		}
	}
	data["Deliveries"] = deliveries

	accounts, err := s.store.ListAccounts(ctx)
	if err != nil {
		return err
	}
	ret, err := s.store.GetRetentionSettings(ctx)
	if err != nil {
		return err
	}
	data["Retention"] = toRetentionView(ret, accounts)
	data["RetError"] = f.retError
	data["RetObs"] = f.retObs
	data["RetDispatch"] = f.retDispatch
	data["RetTranscript"] = f.retTranscript
	return nil
}

func (s *server) fillAPISection(r *http.Request, data map[string]any) error {
	cfg, err := s.store.GetInstanceConfig(r.Context())
	if err != nil {
		return err
	}
	api := map[string]any{"Enabled": cfg.ApiEnabled}
	if cfg.ApiEnabled {
		if cfg.ApiUpdatedBy.Valid {
			if accounts, aerr := s.store.ListAccounts(r.Context()); aerr == nil {
				for _, a := range accounts {
					if a.ID == cfg.ApiUpdatedBy.Int64 {
						api["By"] = a.Username
						break
					}
				}
			}
		}
		if cfg.ApiUpdatedAt.Valid {
			api["At"] = cfg.ApiUpdatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
	}
	data["API"] = api
	return nil
}

func (s *server) fillApertureSection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	editRows, err := s.store.ListVergeCoreFrequencyEditsWithAuthor(ctx)
	if err != nil {
		return err
	}
	editByPort := make(map[uint16]string, len(editRows))
	edits := make([]vergecore.FrequencyEdit, 0, len(editRows))
	for _, e := range editRows {
		editByPort[uint16(e.Port)] = e.Action                                                  // #nosec G115 (DB port written only via 1..65535-validated edit path)
		edits = append(edits, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action}) // #nosec G115 (DB port written only via 1..65535-validated edit path)
	}
	shipped := vergecore.Default()
	// A port you can hide is a signal you can silence (docs/spec/v1-spec.md §3.5).
	effective := shipped.WithFrequencyEdits(edits)
	freq := make([]freqRow, 0, len(effective.FrequencyPairs()))
	for _, p := range effective.FrequencyPairs() {
		action, edited := editByPort[p.Port]
		freq = append(freq, freqRow{
			Port: int(p.Port), AlsoSensitive: shipped.IsSensitive(p),
			Edited: edited, EditAction: action,
		})
	}
	sens := make([]sensRow, 0, len(shipped.SensitivePairs()))
	for _, p := range shipped.SensitivePairs() {
		sens = append(sens, sensRow{
			Port: int(p.Port), Transport: string(p.Transport),
			Service: sensitiveServiceLabels[int(p.Port)],
		})
	}
	c := effective.Count()
	data["Counts"] = c
	data["UDPCount"] = c.UDP
	data["Frequency"] = freq
	data["Sensitive"] = sens
	data["VCError"] = f.vcError
	data["VCPort"] = f.vcPort
	return nil
}

func (s *server) fillTeamSection(r *http.Request, acct db.Account, f settingsForms, data map[string]any) error {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil {
		return err
	}
	members := toAccountRows(accounts, acct.ID)
	data["Members"] = members
	data["TeamError"] = f.teamError
	data["RoleError"] = f.roleError

	find := func(id int64) *accountRow {
		if id == 0 {
			return nil
		}
		for i := range members {
			if members[i].ID == id {
				return &members[i]
			}
		}
		return nil
	}
	// A destructive act opens as a navigation, never a menu click (docs/guides/accounts.md).
	q := r.URL.Query()
	qid := func(key string) int64 {
		if v := q.Get(key); v != "" {
			if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				return id
			}
		}
		return 0
	}

	if m := find(qid("role")); m != nil && !m.IsSelf {
		data["RoleTarget"] = m
	}
	if m := find(qid("reenroll")); m != nil && !m.IsSelf {
		data["ReenrollTarget"] = m
	}
	removeID := f.removeID
	if removeID == 0 {
		removeID = qid("remove")
	}
	if m := find(removeID); m != nil && !m.IsSelf {
		data["RemoveTarget"] = m
		data["RemoveError"] = f.removeError
	}
	data["InviteOpen"] = f.inviteOpen || q.Get("invite") != "" || f.inviteLink != ""
	data["InviteLink"] = f.inviteLink
	data["InviteRole"] = f.inviteRole
	return nil
}

func (s *server) fillAuditSection(_ *http.Request, data map[string]any) error {
	// No queryable log exists, so this ships an empty state, never fabricated data (ADR-0110).
	data["AuditRows"] = nil
	return nil
}

func (s *server) fillSessionsSection(r *http.Request, f settingsForms, data map[string]any) error {
	now := s.now()
	rows, err := s.store.ListAllActiveSessions(r.Context(), pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return err
	}
	curSessionID, haveCurSession := s.currentSessionID(r)
	sessions := make([]sessionRow, 0, len(rows))
	for _, row := range rows {
		sr := sessionRow{
			ID: row.ID, AccountID: row.AccountID, Account: row.Username, Role: row.Role,
			Device: sessionDeviceFromUA(row.UserAgent), IP: row.Ip,
			LastSeen: agoLabel(row.LastSeenAt.Time, now),
			Current:  haveCurSession && row.ID == curSessionID,
		}
		if sr.IP == "" {
			sr.IP = "—"
		}
		sessions = append(sessions, sr)
	}
	data["Sessions"] = sessions

	q := r.URL.Query()
	if v := q.Get("revoke"); v != "" {
		if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
			for i := range sessions {
				if sessions[i].ID == id {
					data["RevokeSessionTarget"] = &sessions[i]
					break
				}
			}
		}
	}
	revokeAcct := f.revokeAccountID
	if revokeAcct == 0 {
		if v := q.Get("revoke-account"); v != "" {
			if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				revokeAcct = id
			}
		}
	}
	if revokeAcct != 0 {
		for i := range sessions {
			if sessions[i].AccountID == revokeAcct {
				data["RevokeAccountTarget"] = map[string]any{
					"AccountID": revokeAcct, "Username": sessions[i].Account,
				}
				data["RevokeAccountError"] = f.revokeAccountError
				break
			}
		}
	}
	return nil
}

func (s *server) revokeSessionAdmin(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.backToSection(w, r, "sessions")
		return
	}
	// Deliberately not owner-scoped: this ends any account's session, unlike Profile's (#407).
	if err := s.store.RevokeSessionByIDForAdmin(r.Context(), db.RevokeSessionByIDForAdminParams{
		ID: id, RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "admin revoke session", err)
		return
	}
	s.flashSettings(w, r, settingsForms{
		section: "sessions",
		notice:  "Session revoked — its next request lands on the sign-in screen.",
	})
}

func (s *server) revokeAccountSessions(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("account_id"), 10, 64)
	if err != nil {
		s.backToSection(w, r, "sessions")
		return
	}
	target, err := s.store.GetAccountByID(r.Context(), id)
	if err != nil {
		s.backToSection(w, r, "sessions")
		return
	}
	if strings.TrimSpace(r.FormValue("confirm_name")) != target.Username {
		s.failSettings(w, r, settingsForms{
			section: "sessions", revokeAccountID: id,
			revokeAccountError: "That did not match. Type " + target.Username + " exactly to revoke every session.",
		})
		return
	}
	if err := s.store.RevokeAllSessionsForAccount(r.Context(), db.RevokeAllSessionsForAccountParams{
		AccountID: id, RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "admin revoke account sessions", err)
		return
	}
	s.flashSettings(w, r, settingsForms{
		section: "sessions",
		notice:  "Every session for that account was revoked.",
	})
}

func sessionDeviceFromUA(ua string) string {
	if ua == "" {
		return "Unknown device"
	}
	if strings.HasPrefix(ua, "verge-cli") {
		if i := strings.IndexByte(ua, '('); i >= 0 {
			if j := strings.IndexByte(ua[i:], ')'); j > 1 {
				if host := strings.TrimSpace(ua[i+1 : i+j]); host != "" {
					return "CLI · " + host
				}
			}
		}
		return "CLI"
	}
	browser := "Browser"
	switch {
	case strings.Contains(ua, "Firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "Edg"):
		browser = "Edge"
	case strings.Contains(ua, "Chrome"), strings.Contains(ua, "Chromium"):
		browser = "Chrome"
	case strings.Contains(ua, "Safari"):
		browser = "Safari"
	}
	os := ""
	switch {
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		os = "macOS"
	case strings.Contains(ua, "Windows"):
		os = "Windows"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"):
		os = "iOS"
	case strings.Contains(ua, "Android"):
		os = "Android"
	case strings.Contains(ua, "Linux"):
		os = "Linux"
	}
	if os != "" {
		return browser + " · " + os
	}
	return browser
}

func (s *server) fillInstanceSection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	inst := map[string]any{
		"License":    "AGPL-3.0 · self-hosted",
		"Uptime":     humanizeDuration(s.now().Sub(s.startedAt)),
		"Version":    s.buildVersion(),
		"QueueDepth": "",
		"DiskPct":    0,
		"DiskDetail": "",
		"PgLabel":    "postgres",
		"PgDetail":   "",
	}

	if rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit); err == nil {
		var waiting int64
		for _, row := range rows {
			waiting += row.Ready + row.Running
		}
		inst["QueueDepth"] = waiting
	} else {
		log.Printf("web: instance: queue depth: %v", err)
	}

	if used, total, ok := diskUsage("."); ok {
		inst["DiskDetail"] = diskLabel(used, total)
		inst["DiskPct"] = int(used * 100 / total) // #nosec G115 -- used<=total (guarded in diskUsage), so the percentage is 0..100
	}

	if h, err := s.store.GetInstanceHealth(ctx); err == nil {
		inst["PgLabel"] = pgLabel(h.ServerVersion)
		inst["PgDetail"] = humanBytes(h.DbSizeBytes)
	} else {
		log.Printf("web: instance: db health: %v", err)
	}

	var fleet []map[string]any
	if rows, err := s.store.ListVantages(r.Context()); err == nil {
		for _, v := range rows {
			avail := v.Availability.String
			if avail == "" {
				avail = "pending"
			}
			fleet = append(fleet, map[string]any{
				"Name": v.Name, "Latency": vantageLatencyLabel(v.LatencyMs), "Avail": avail,
			})
		}
	} else {
		log.Printf("web: instance: list vantages: %v", err)
	}
	inst["Vantages"] = fleet

	if pending, ok := s.migrationsPending(ctx); ok {
		inst["Migrations"] = map[string]any{"Pending": pending}
	}

	if cfg, err := s.store.GetInstanceConfig(ctx); err == nil {
		release := map[string]any{
			"CheckEnabled": cfg.UpdateCheckEnabled,
			"Steps":        updateHostSteps,
		}
		state := "disabled"
		if cfg.UpdateCheckEnabled {
			state = "current"
			if cfg.ReleaseState.String == "newer" {
				state = "newer"
				release["Latest"] = map[string]any{
					"Version": cfg.ReleaseLatestVersion.String,
					"Notes":   cfg.ReleaseLatestNotes.String,
				}
			}
			if cfg.ReleaseCheckedAt.Valid {
				release["CheckedAt"] = cfg.ReleaseCheckedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
			}
		}
		release["State"] = state
		inst["Release"] = release

		backup := map[string]any{"LastAt": "", "LastSize": ""}
		if cfg.LastBackupAt.Valid {
			backup["LastAt"] = cfg.LastBackupAt.Time.UTC().Format("2006-01-02 15:04 UTC")
			backup["LastSize"] = humanBytes(cfg.LastBackupSize.Int64)
		}
		inst["Backup"] = backup
	} else {
		log.Printf("web: instance: release config: %v", err)
	}

	if f.restoreError != "" {
		inst["RestoreError"] = f.restoreError
	}
	if f.preflight != nil {
		inst["Preflight"] = map[string]any{
			"File": f.preflight.File, "TakenAt": f.preflight.TakenAt,
			"Subjects": f.preflight.Subjects, "Schema": f.preflight.Schema,
		}
	}
	if f.restoreConfirm != nil {
		inst["RestoreConfirm"] = map[string]any{
			"File": f.restoreConfirm.File, "TakenAt": f.restoreConfirm.TakenAt,
			"Subjects": f.restoreConfirm.Subjects,
		}
	}

	data["Instance"] = inst
	return nil
}

// A feed-delivered step list would put arbitrary shell text in front of an admin (ADR-0124).

var updateHostSteps = []string{
	"# on the host — verge cannot rewrite its own image",
	"docker compose pull",
	"docker compose up -d web worker",
	"docker compose ps web worker",
}

func (s *server) migrationsPending(ctx context.Context) (int, bool) {
	if s.pool == nil {
		return 0, false
	}
	var applied int64
	if err := s.pool.QueryRow(ctx, "SELECT COALESCE(max(version_id), 0) FROM goose_db_version").Scan(&applied); err != nil {
		log.Printf("web: instance: migrations applied: %v", err)
		return 0, false
	}
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		log.Printf("web: instance: migrations embed: %v", err)
		return 0, false
	}
	pending := 0
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if v, ok := migrationVersion(name); ok && v > applied {
			pending++
		}
	}
	return pending, true
}

func migrationVersion(name string) (int64, bool) {
	i := strings.IndexByte(name, '_')
	if i <= 0 {
		return 0, false
	}
	v, err := strconv.ParseInt(name[:i], 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func (s *server) updateCheckToggle(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// While off the worker dispatches no check, so an air-gapped install stays silent (ADR-0124).
	enabled := r.FormValue("enabled") == "true"
	if err := s.store.SetUpdateCheckEnabled(r.Context(), db.SetUpdateCheckEnabledParams{
		UpdateCheckEnabled:   enabled,
		UpdateCheckUpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "set update check enabled", err)
		return
	}
	s.backToSection(w, r, "instance")
}

func (s *server) apiToggle(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// The surface is read-only always: there is no write half a flip could enable (ADR-0123).
	enabled := r.FormValue("enabled") == "true"
	if err := s.store.SetAPIEnabled(r.Context(), db.SetAPIEnabledParams{
		ApiEnabled:   enabled,
		ApiUpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "set api enabled", err)
		return
	}
	if enabled {
		s.toastRedirectBack(w, r, "/settings?tab=api", "ok", "API access enabled",
			"Personal tokens now answer GET /api/v1/… — read-only, always.")
		return
	}
	s.toastRedirectBack(w, r, "/settings?tab=api", "neutral", "API access disabled", "")
}

func diskLabel(used, total uint64) string {
	const gb = 1 << 30
	return fmt.Sprintf("%.1f / %.0f GB", float64(used)/gb, float64(total)/gb)
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return strconv.FormatInt(n, 10) + " B"
	}
}

func pgLabel(version string) string {
	major := strings.TrimSpace(version)
	if i := strings.IndexFunc(major, func(r rune) bool { return r < '0' || r > '9' }); i >= 0 {
		major = major[:i]
	}
	if major == "" {
		return "postgres"
	}
	return "postgres " + major
}

func humanizeDuration(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d"
	case d >= time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	case d >= time.Minute:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	default:
		return strconv.Itoa(int(d/time.Second)) + "s"
	}
}

func toAccountRows(rows []db.ListAccountsRow, selfID int64) []accountRow {
	out := make([]accountRow, 0, len(rows))
	for _, a := range rows {
		r := accountRow{
			ID: a.ID, Username: a.Username, Initials: initialsFromUsername(a.Username),
			Role: a.Role, TotpEnabled: a.TotpEnabled, IsSelf: a.ID == selfID,
		}
		if a.CreatedAt.Valid {
			r.At = a.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, r)
	}
	return out
}

func toChannelViews(rows []db.ListChannelsRow) []channelView {
	out := make([]channelView, 0, len(rows))
	for _, c := range rows {
		v := channelView{
			ID: c.ID, URL: c.Url, Drift: c.RouteDrift, Coverage: c.RouteCoverage,
			Clock: c.RouteClock, Enabled: c.Enabled, HasSecret: c.HasSecret,
			By: c.CreatedByUsername,
		}
		checked := map[string]bool{"drift": c.RouteDrift, "coverage": c.RouteCoverage, "clock": c.RouteClock}
		for _, name := range channelClasses {
			v.ClassStates = append(v.ClassStates, classState{Name: name, Checked: checked[name]})
			if checked[name] {
				v.Classes = append(v.Classes, name)
			}
		}
		if c.CreatedAt.Valid {
			v.At = c.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
		}
		out = append(out, v)
	}
	return out
}

func toRetentionView(ret db.GetRetentionSettingsRow, accounts []db.ListAccountsRow) retentionView {
	v := retentionView{
		ObservationCurrencyDays: ret.ObservationCurrencyDays,
		DispatchCadenceMultiple: ret.DispatchCadenceMultiple,
		TranscriptCurrencyDays:  ret.TranscriptCurrencyDays,
	}
	if ret.UpdatedBy.Valid {
		for _, a := range accounts {
			if a.ID == ret.UpdatedBy.Int64 {
				v.UpdatedBy = a.Username
				break
			}
		}
	}
	if ret.UpdatedAt.Valid {
		v.UpdatedAt = ret.UpdatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
	}
	return v
}

// Only an address scope is its own enumeration, so only these tiers walk per address (ADR-0047).

var addressScopeScanKinds = map[string]bool{"hot": true, "cold": true}

func toAddressCapView(cfg db.GetInstanceConfigRow, scans []db.Scan, accounts []db.ListAccountsRow) addressCapView {
	// The same fallback the declaration path applies, so the readout matches what it checks.
	capVal := cfg.SeedAddressCap
	if capVal <= 0 {
		capVal = int64(seed.DefaultAddressCap)
	}
	probes := commaGroup(strconv.FormatInt(capVal, 10))
	v := addressCapView{
		Cap:            capVal,
		CapLabel:       probes,
		LargestScopeV4: fmt.Sprintf("/%d", seed.LargestPrefixLen(int(capVal), 32)),
		LargestScopeV6: fmt.Sprintf("/%d", seed.LargestPrefixLen(int(capVal), 128)),
		DiskProjection: projectedEvidentialDiskPerYear(capVal),
	}
	for _, sc := range scans {
		if !sc.Enabled || !addressScopeScanKinds[sc.Kind] {
			continue
		}
		line := capSweepLine{
			Scan:    sc.Kind,
			Cadence: cadenceLabel(sc.CadenceSeconds),
			Probes:  probes,
		}
		if ports := addressScopePorts[sc.Kind]; ports > 0 {
			eff := effectiveCadenceSeconds(capVal, ports)
			line.Effective = projectedPassLabel(eff)
			line.Outpaces = sc.CadenceSeconds > 0 && eff > float64(sc.CadenceSeconds)
		}
		v.SweepLoad = append(v.SweepLoad, line)
	}
	if cfg.SeedAddressCapUpdatedBy.Valid {
		for _, a := range accounts {
			if a.ID == cfg.SeedAddressCapUpdatedBy.Int64 {
				v.UpdatedBy = a.Username
				break
			}
		}
	}
	if cfg.SeedAddressCapUpdatedAt.Valid {
		v.UpdatedAt = cfg.SeedAddressCapUpdatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
	}
	return v
}

func projectedEvidentialDiskPerYear(addrCap int64) string {
	// 13 GB a year per declared /22 is ADR-0041's measured grounding, scaled linearly here.
	gbPerYear := 13.0 * float64(addrCap) / 1024.0
	switch {
	case gbPerYear >= 1024:
		return fmt.Sprintf("≈ %.1f TB / year", gbPerYear/1024)
	case gbPerYear >= 10:
		return fmt.Sprintf("≈ %.0f GB / year", gbPerYear)
	case gbPerYear >= 1:
		return fmt.Sprintf("≈ %.1f GB / year", gbPerYear)
	default:
		return fmt.Sprintf("≈ %.0f MB / year", gbPerYear*1024)
	}
}

// 131 is verge-core's shipped TCP count; edits move it (docs/spec/v1-spec.md §3.5).

var addressScopePorts = map[string]int64{"hot": 131, "cold": 65535}

// The cap has no ceiling, so an int64 product would overflow; this is a stated projection.

func effectiveCadenceSeconds(addresses, portsPerAddress int64) float64 {
	p := connectoutcome.DefaultProfile()
	rate := float64(p.PerVantagePacketsPerSec)
	if rate <= 0 {
		rate = 1
	}
	// Counting every retry keeps the worst-case figure from understating the lag (ADR-0127).
	attempts := float64(1 + p.Retries)
	return float64(addresses) * float64(portsPerAddress) * attempts / rate
}

func projectedPassLabel(seconds float64) string {
	switch {
	case seconds < 60:
		return "≈ under a minute"
	case seconds < 3600:
		return fmt.Sprintf("≈ %.0f min", math.Round(seconds/60))
	case seconds < 86400:
		return fmt.Sprintf("≈ %.0f h", math.Round(seconds/3600))
	case seconds < 30*86400:
		return fmt.Sprintf("≈ %.0f days", math.Round(seconds/86400))
	case seconds < 365*86400:
		return fmt.Sprintf("≈ %.0f months", math.Round(seconds/(30*86400)))
	default:
		return fmt.Sprintf("≈ %.1f years", seconds/(365*86400))
	}
}

func classesFromForm(r *http.Request) (drift, coverage, clock bool) {
	// Routing keys on the class and nothing finer; a per-cause predicate is refused (ADR-0091).
	return r.FormValue("drift") != "", r.FormValue("coverage") != "", r.FormValue("clock") != ""
}

func optionalSecret(v string) pgtype.Text {
	if strings.TrimSpace(v) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

func validateChannelURL(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return "", "Enter an absolute URL, like https://example.com/hook."
	}
	switch u.Scheme {
	case "https":
		// TLS encrypts the hop but stops no admin aiming a channel at an internal service (#325).
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && custody.IsNonGloballyReachable(ip) {
			return "", "That host is an internal address; a channel must point at a public https endpoint."
		}
		return u.String(), ""
	case "http":
		// Only an address literal earns the plaintext exemption (notification-channels.md §4.1).
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && ip.IsLoopback() {
			return u.String(), ""
		}
		return "", "http:// is allowed only to a loopback address literal; use https://."
	default:
		return "", "The URL must be https:// (or http:// to a loopback address)."
	}
}
