package main

import (
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

// The Settings screen is the operator's dials (v1 spec §6.1): the tabbed console
// destination ported from examples/console/Settings.jsx. It folds today's
// settings (accounts, channels, retention), the scans monitor, the vantages the
// worker measures from, the message panel, the verge-core port set, the delivery
// record, and source enablement into seven query-param sub-tabs
// (/settings?tab=<id>). Every mutation it hosts is an authenticated admin act
// (§4.3), reached only through requireAdmin; the folded read surfaces (/scans,
// /messages, /verge-core, /sources) render one section for a viewer.

// channelView is a declared Channel shaped for rendering. It never carries the
// secret, only whether one is set: the secret is write-only and the render path
// is structurally unable to hold it (CONTEXT.md "Channel").
type channelView struct {
	ID        int64
	URL       string
	Drift     bool
	Coverage  bool
	Clock     bool
	Enabled   bool
	HasSecret bool
	By        string
	At        string
}

// accountRow is one account in the management list. It carries no password hash
// and no TOTP secret — managing an account needs neither.
type accountRow struct {
	ID          int64
	Username    string
	Role        string
	TotpEnabled bool
	At          string
	IsSelf      bool
}

// retentionView renders the two dials and who last moved them.
type retentionView struct {
	ObservationCurrencyDays int64
	DispatchCadenceMultiple int64
	UpdatedBy               string
	UpdatedAt               string
}

// vantageRow is one measurement position shaped for the vantages section. A
// vantage is never a probe/scanner/agent (CONTEXT.md): the render carries only its
// measurement name, verified class, availability, resolver and endpoint — never a
// private key.
type vantageRow struct {
	Name         string
	Class        string
	Availability string
	Resolver     string
	Endpoint     string
}

// settingsForms carries the echo state of the Settings screen's forms so a
// rejected submission on one section leaves its own error and typed values in
// place without disturbing the others. section names the section that failed and
// drives the response status; tab forces the active sub-tab (a folded read
// surface renders one section by name), and notice carries a success line.
type settingsForms struct {
	section string // "", "team", "channels", "retention" or "vergecore"
	tab     string // explicit active tab; when "", derived from section (default scans)
	notice  string // a success line, rendered above the active section

	// team (T18). teamError is an inline error on the members surface; roleError is
	// the change-role guard's message. inviteLink is a freshly minted join URL,
	// revealed once by createInvite; inviteOpen re-opens the invite dialog on a
	// rejected mint and inviteRole echoes its role. removeID/removeError re-open the
	// remove ConfirmDialog on a typed-name mismatch or a guard refusal.
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

	// sso (#293). ssoError is an inline error on the single-sign-on surface; the
	// remaining fields echo a rejected add-provider form back so the operator does not
	// retype it (the add form renders unconditionally, so no open/closed flag is
	// needed).
	ssoError    string
	ssoSlug     string
	ssoName     string
	ssoIssuer   string
	ssoClientID string

	retError    string
	retObs      string
	retDispatch string

	vcError string
	vcPort  string
}

// settingsTabs is the sub-tab order of the Settings screen, ported from
// examples/console/Settings.jsx's SettingsNav groups: Scanning (scans, vantages),
// Access (single sign-on, team, audit log), Discovery (sources, port aperture),
// Instance (health), then Delivery (channels, integrations, messages, delivery
// record). Each is reached at /settings?tab=<id>.
var settingsTabs = []string{
	"scans", "vantages",
	"sso", "team", "audit",
	"sources", "aperture",
	"instance",
	"channels", "integrations", "messages", "delivery",
}

// validTab keeps the query param to a known section, defaulting to the first.
// The pre-V3 "access" tab split into "sso" and "team" (T18); a lingering
// tab=access link (a bookmark, or the /account redirect before it was retargeted)
// lands on Team, where account management now lives, rather than 404-ing.
func validTab(t string) string {
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

// tabForSection maps a failing form's section to the tab that hosts it, so a
// rejected submission re-renders with its own section active.
func tabForSection(section string) string {
	switch section {
	case "sso":
		return "sso"
	case "team":
		return "team"
	case "channels":
		return "channels"
	case "retention":
		return "delivery"
	case "vergecore":
		return "aperture"
	default:
		return "scans"
	}
}

func (s *server) settingsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: validTab(r.URL.Query().Get("tab"))})
}

// --- team ------------------------------------------------------------------

// inviteTTL bounds a Team invite's life, matching the Settings.jsx invite dialog's
// "expires in 7 days". A join link older than this is refused at /invite (T19's
// lookupInvite), so a leaked-then-stale link is inert.
const inviteTTL = 7 * 24 * time.Hour

// inviteAccount mints a single-use invite from the Settings → Team dialog and
// reveals the join link once (T18). It is the CREATION side of the invite table
// T19 shipped for acceptance: unlike the pre-V3 path it never creates an account
// directly — the invitee chooses their own username and password at /invite, and
// the role applies on acceptance. Accounts on this build are usernames with no
// identity provider, so the invite binds a role, not an address: the dialog asks
// only the role and the plaintext token rides one URL handed out of band (also
// written to the web logs, exactly as the setup and reset tokens are).
func (s *server) inviteAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	role := r.FormValue("role")
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
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
	// The one delivery this self-hosted build honestly has: the operator's own logs.
	log.Printf("web: invite minted at role %q; accept it at %s (expires in %s)", role, link, inviteTTL)
	s.renderSettings(w, r, acct, settingsForms{tab: "team", inviteLink: link})
}

// inviteLink builds the absolute join URL an invitee presents at /invite. It reads
// the request host (never a proxy-forwarding header) and infers the scheme from the
// TLS state or the secure-cookie flag, so the copied link works behind a TLS proxy.
func (s *server) inviteLink(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil || s.secureCookies {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/invite?token=" + token
}

// setAccountRole reassigns an account's role. It refuses to demote the last
// admin: an operator must never be able to strip the final admin and lock every
// remaining account out of every mutation. The Save control is disabled until the
// selected role differs (Settings.jsx), so a no-op save never reaches here in the
// UI; the guards still hold on the raw POST.
func (s *server) setAccountRole(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "team", roleError: msg})
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
	http.Redirect(w, r, "/settings?tab=team", http.StatusSeeOther)
}

// reenrollAccount clears a member's second factor (Settings.jsx "Require
// re-enrollment"): their current authenticator stops working at once and the next
// sign-in walks them through TOTP setup again. It never touches a password or a
// session, and it is a no-op guard against a missing row rather than a 500.
func (s *server) reenrollAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "team", teamError: "That account could not be found."})
		return
	}
	if err := s.store.ResetAccountTOTP(r.Context(), id); err != nil {
		s.serverError(w, "reset account totp", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=team", http.StatusSeeOther)
}

// removeAccount removes a member through a typed-name gate — the worst destructive
// act on the Team surface, so the operator must type the member's exact username to
// confirm, and it is reached only through the remove ConfirmDialog (a POST), never a
// menu click. It refuses to remove yourself or the last admin. An account that
// authored attributed acts (a NOT NULL created_by) is refused by the FK with a clear
// message rather than a 500, so its work is never orphaned.
func (s *server) removeAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "team", teamError: "That account could not be found."})
		return
	}
	reopen := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "team", removeID: id, removeError: msg})
	}
	if id == acct.ID {
		// Self has no remove dialog (a member never acts on their own row), so the
		// refusal shows inline rather than in a dialog that would not render.
		s.renderSettings(w, r, acct, settingsForms{section: "team", teamError: "You cannot remove your own account."})
		return
	}
	target, err := s.store.GetAccountByID(r.Context(), id)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "team", teamError: "That account could not be found."})
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
	if err := s.store.DeleteAccount(r.Context(), id); err != nil {
		if isForeignKeyViolation(err) {
			reopen(target.Username + " has declared scopes, channels, or other attributed acts and cannot be removed — reassign or keep the account.")
			return
		}
		s.serverError(w, "delete account", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=team", http.StatusSeeOther)
}

// --- channels --------------------------------------------------------------

// createChannel declares where Messages go. It persists the fields only — the
// outbound POST and the Delivery record land with ticket 27, and nothing here
// delivers anything.
func (s *server) createChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	rawURL := strings.TrimSpace(r.FormValue("url"))
	drift, coverage, clock := classesFromForm(r)
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
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
	http.Redirect(w, r, "/settings?tab=channels", http.StatusSeeOther)
}

// updateChannel edits a channel's URL, routing classes and enabled state, and
// applies a secret change if one was asked for. The secret is write-only: a
// blank secret field leaves the stored one untouched, the clear box removes it,
// and a value replaces it.
func (s *server) updateChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "channels", chanError: msg})
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
	// The secret write is separate so leaving the field blank keeps the current
	// one. The clear box wins over any typed value.
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
	http.Redirect(w, r, "/settings?tab=channels", http.StatusSeeOther)
}

// deleteChannel removes a channel. It is idempotent: deleting a row that is
// already gone satisfies the operator's intent either way.
func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		s.renderSettings(w, r, acct, settingsForms{section: "channels", chanError: "That channel could not be found."})
		return
	}
	if err := s.store.DeleteChannel(r.Context(), id); err != nil {
		s.serverError(w, "delete channel", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=channels", http.StatusSeeOther)
}

// --- retention -------------------------------------------------------------

// updateRetention persists the two dial values. Both are floored, and both floors
// are DERIVED not asserted (ADR-0094) — never presented as an operator choice. The
// observation-currency dial (#208, §4.6) floors at the tightest observation bound
// in force: k cadences of the tightest enabled Scan, below which the control
// changes no row at all. The Dispatch dial (#209, §4.6) floors at k cadences of the
// slowest enabled Scan. For both, 0 is the unbounded v1 default and always allowed,
// and any positive value below the floor is rejected. Deletion of expired rows is a
// structurally separate path (internal/retention), never reached from here.
func (s *server) updateRetention(w http.ResponseWriter, r *http.Request, acct db.Account) {
	obsRaw := strings.TrimSpace(r.FormValue("observation_currency_days"))
	dispRaw := strings.TrimSpace(r.FormValue("dispatch_cadence_multiple"))
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
			section: "retention", retError: msg, retObs: obsRaw, retDispatch: dispRaw,
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
	// The observation floor is the tightest bound in force — k cadences of the
	// tightest enabled Scan. The query still reads each row's own bound; this only
	// forbids the operator naming a dial the whole corpus already outlives.
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
		UpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update retention", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=delivery", http.StatusSeeOther)
}

// --- render ----------------------------------------------------------------

// renderSettings assembles the active sub-tab and renders the tabbed Settings
// page. It gathers only the data the active section needs, so a folded read
// surface pays for its own section alone. A failing form re-renders its own tab
// with the echo state and a 400.
func (s *server) renderSettings(w http.ResponseWriter, r *http.Request, acct db.Account, f settingsForms) {
	active := f.tab
	if active == "" {
		active = tabForSection(f.section)
	}

	// The Integrations surface is hidden (#388, integrationsEnabled). A direct
	// ?tab=integrations navigation renders no tab and no section, so bounce it to
	// the default Scans tab rather than an empty page — nothing integration-related
	// is reachable while the flag is off.
	if active == "integrations" && !integrationsEnabled {
		http.Redirect(w, r, "/settings?tab=scans", http.StatusSeeOther)
		return
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
		err = s.fillScansSection(r, acct, data)
	case "vantages":
		err = s.fillVantagesSection(r, data)
	case "sso":
		err = s.fillSSOSection(r, f, data)
	case "team":
		err = s.fillTeamSection(r, acct, f, data)
	case "audit":
		err = s.fillAuditSection(r, data)
	case "sources":
		err = s.fillSourcesSection(r, data)
	case "aperture":
		err = s.fillApertureSection(r, f, data)
	case "instance":
		err = s.fillInstanceSection(r, data)
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

	status := http.StatusOK
	if f.section != "" {
		status = http.StatusBadRequest
	}
	s.renderStatus(w, status, "settings", data)
}

// fillVantagesSection lists the provisioned measurement positions (CONTEXT.md
// "Vantage"). A read-only display: provisioning lives on Scope, and a vantage is
// never a probe/scanner/agent here.
func (s *server) fillVantagesSection(r *http.Request, data map[string]any) error {
	rows, err := s.store.ListVantages(r.Context())
	if err != nil {
		return err
	}
	out := make([]vantageRow, 0, len(rows))
	for _, v := range rows {
		vr := vantageRow{
			Name: v.Name, Class: v.Class, Availability: v.Availability.String,
			Resolver: v.Resolver, Endpoint: endpointString(v.Host.String, v.Port.Int32),
		}
		if vr.Availability == "" {
			vr.Availability = "pending"
		}
		if vr.Resolver == "" {
			vr.Resolver = "—"
		}
		out = append(out, vr)
	}
	data["Vantages"] = out
	// The prober-provisioning read (Settings.jsx ProberProvision): each provisioned
	// vantage's published PUBLIC key (reveal-once at first render, never a private
	// key), its host-key pin status (the value never reaches web), and its egress. The
	// provisioning ACT lives on Scope (POST /probers); this tab renders the read.
	data["Probers"] = toProberViews(rows)
	return nil
}

// fillChannelsSection reads the declared channels and the create-form echo.
func (s *server) fillChannelsSection(r *http.Request, f settingsForms, data map[string]any) error {
	channels, err := s.store.ListChannels(r.Context())
	if err != nil {
		return err
	}
	// The create-channel form defaults to all three classes; only a rejected
	// create echoes the operator's own selection back.
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
	return nil
}

// fillDeliverySection carries the operational-record group: the delivery outcomes
// register (ADR-0039/ADR-0081) and the two retention dials. The verge-core hot port
// set moved to its own Port-aperture tab under Discovery (T18, matching
// Settings.jsx's SettingsNav) — see fillApertureSection.
func (s *server) fillDeliverySection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()

	// Delivery outcomes register — real outcomes, host-only so no embedded token
	// leaks (message.go's toDeliveryView). Best-effort: a read failure degrades to
	// an empty register rather than 500ing the whole section.
	var deliveries []deliveryView
	if outcomes, derr := s.store.ListDeliveryOutcomes(ctx); derr == nil {
		for _, o := range outcomes {
			deliveries = append(deliveries, toDeliveryView(o))
		}
	}
	data["Deliveries"] = deliveries

	// Retention dials.
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
	return nil
}

// fillApertureSection carries the verge-core hot port set (§3.5, Settings.jsx's
// ApertureSection): the release-authored sensitive tier rendered read-only, and the
// operator-editable frequency tier. A frequency edit is stored as a delta over the
// shipped default and applied at hot fan-out, so the sensitive tier is unreachable
// from every write path by construction — a port you can hide is a signal you can
// silence.
func (s *server) fillApertureSection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	editRows, err := s.store.ListVergeCoreFrequencyEditsWithAuthor(ctx)
	if err != nil {
		return err
	}
	editByPort := make(map[uint16]string, len(editRows))
	edits := make([]vergecore.FrequencyEdit, 0, len(editRows))
	for _, e := range editRows {
		editByPort[uint16(e.Port)] = e.Action
		edits = append(edits, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action})
	}
	shipped := vergecore.Default()
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
		sens = append(sens, sensRow{Port: int(p.Port), Transport: string(p.Transport)})
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

// fillTeamSection carries the Team surface (Settings.jsx TeamSection): the members
// list, the two-role explainer, and the change-role / require-re-enrollment / remove
// / invite dialogs. Each dialog is opened by a query param so the destructive ones
// are a navigation, never a menu click that fires the act; a rejected POST re-opens
// its own dialog through settingsForms. The two roles are admin and viewer only —
// there is no operator role.
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
	q := r.URL.Query()
	qid := func(key string) int64 {
		if v := q.Get(key); v != "" {
			if id, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				return id
			}
		}
		return 0
	}

	// Change-role dialog: opened by ?role=<id>. A member can never act on their own
	// row, so a role param naming self renders no dialog.
	if m := find(qid("role")); m != nil && !m.IsSelf {
		data["RoleTarget"] = m
	}
	// Require-re-enrollment dialog: opened by ?reenroll=<id>.
	if m := find(qid("reenroll")); m != nil && !m.IsSelf {
		data["ReenrollTarget"] = m
	}
	// Remove ConfirmDialog: opened by ?remove=<id> (GET) or re-opened by a rejected
	// POST through f.removeID, which also carries the typed-name mismatch message.
	removeID := f.removeID
	if removeID == 0 {
		removeID = qid("remove")
	}
	if m := find(removeID); m != nil && !m.IsSelf {
		data["RemoveTarget"] = m
		data["RemoveError"] = f.removeError
	}
	// Invite dialog: opened by ?invite=1 (GET), re-opened on a rejected mint, or shown
	// with the freshly minted join link on success (revealed once).
	data["InviteOpen"] = f.inviteOpen || q.Get("invite") != "" || f.inviteLink != ""
	data["InviteLink"] = f.inviteLink
	data["InviteRole"] = f.inviteRole
	return nil
}

// fillAuditSection renders the audit-log tab. This build keeps no queryable log of
// admin acts — source enablement, for one, "keeps no log line of its own" and is
// dated only by the batch it moves — so the honest state is empty rather than a
// fabricated feed. The delivery record (Delivery tab) and the message store are the
// operational records that do exist; this names them.
func (s *server) fillAuditSection(_ *http.Request, data map[string]any) error {
	data["AuditRows"] = nil
	return nil
}

// fillInstanceSection carries the instance-health tab (Settings.jsx InstanceSection)
// as real reads only — no fabricated version string, uptime figure, or queue depth
// where the datum does not exist. What is real: the licence/build stance (AGPL-3.0,
// self-hosted), the process uptime since start, that Postgres answered this render,
// and the provisioned vantage fleet with each vantage's availability.
func (s *server) fillInstanceSection(r *http.Request, data map[string]any) error {
	data["Licence"] = "AGPL-3.0 · self-hosted"
	data["Uptime"] = humanizeDuration(s.now().Sub(s.startedAt))

	var fleet []vantageRow
	if rows, err := s.store.ListVantages(r.Context()); err == nil {
		for _, v := range rows {
			avail := v.Availability.String
			if avail == "" {
				avail = "pending"
			}
			fleet = append(fleet, vantageRow{Name: v.Name, Class: v.Class, Availability: avail})
		}
	} else {
		log.Printf("web: instance: list vantages: %v", err)
	}
	data["Fleet"] = fleet
	return nil
}

// humanizeDuration renders a process uptime as a terse figure (e.g. 41d, 6h, 12m,
// 8s), the shape Settings.jsx's uptime stat shows. Anything under a minute reads in
// seconds so a freshly started instance never renders a bare 0.
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
			ID: a.ID, Username: a.Username, Role: a.Role,
			TotpEnabled: a.TotpEnabled, IsSelf: a.ID == selfID,
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

// --- helpers ---------------------------------------------------------------

// classesFromForm reads the three routing-class checkboxes. Routing is by class
// and nothing finer (ADR-0091); an absent box is that class switched off.
func classesFromForm(r *http.Request) (drift, coverage, clock bool) {
	return r.FormValue("drift") != "", r.FormValue("coverage") != "", r.FormValue("clock") != ""
}

// optionalSecret maps a submitted secret to a nullable column: blank means no
// secret, anything else is stored verbatim (write-only, never rendered back).
func optionalSecret(v string) pgtype.Text {
	if strings.TrimSpace(v) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: v, Valid: true}
}

// validateChannelURL accepts an absolute https URL, and http only to a loopback
// address literal (notification-channels.md §4.1: http is refused at
// configuration time except to loopback, tested over the address). A loopback
// hostname is not accepted here — resolving a name to confirm it is loopback is
// delivery-time work that lands with ticket 27; until then only an unambiguous
// loopback literal earns the plaintext exemption. It returns the normalised URL
// and an empty message on success, or "" and a user-facing message.
//
// An https URL whose host is an IP LITERAL in a non-globally-reachable range —
// loopback, link-local (incl. the 169.254.169.254 cloud-metadata address),
// RFC1918/ULA private space, and the rest of the special-purpose registry — is
// refused here too (#325): the transport encrypts the hop but does nothing to
// stop a settings admin pointing a channel at an internal service and having the
// worker POST the signed body to it (config SSRF). A host given as a NAME is not
// resolved here — that is delivery-time work (the runner re-checks the resolved
// address before every POST), so this layer bars only the unambiguous literal.
func validateChannelURL(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return "", "Enter an absolute URL, like https://example.com/hook."
	}
	switch u.Scheme {
	case "https":
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && custody.IsNonGloballyReachable(ip) {
			return "", "That host is an internal address; a channel must point at a public https endpoint."
		}
		return u.String(), ""
	case "http":
		if ip, err := netip.ParseAddr(u.Hostname()); err == nil && ip.IsLoopback() {
			return u.String(), ""
		}
		return "", "http:// is allowed only to a loopback address literal; use https://."
	default:
		return "", "The URL must be https:// (or http:// to a loopback address)."
	}
}
