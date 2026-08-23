package main

import (
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

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
	section string // "", "accounts", "channels", "retention" or "vergecore"
	tab     string // explicit active tab; when "", derived from section (default scans)
	notice  string // a success line, rendered above the active section

	acctError    string
	acctUsername string
	acctRole     string
	roleError    string

	chanError    string
	chanURL      string
	chanDrift    bool
	chanCoverage bool
	chanClock    bool

	retError    string
	retObs      string
	retDispatch string

	vcError string
	vcPort  string
}

// settingsTabs is the sub-tab order of the Settings screen, ported from
// examples/console/Settings.jsx: the two Scanning sections, the delivery group,
// then Access. Each is reached at /settings?tab=<id>.
var settingsTabs = []string{"scans", "vantages", "sources", "channels", "messages", "delivery", "access", "integrations"}

// validTab keeps the query param to a known section, defaulting to the first.
func validTab(t string) string {
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
	case "accounts":
		return "access"
	case "channels":
		return "channels"
	case "retention", "vergecore":
		return "delivery"
	default:
		return "scans"
	}
}

func (s *server) settingsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{tab: validTab(r.URL.Query().Get("tab"))})
}

// --- accounts --------------------------------------------------------------

// inviteAccount creates an account from the Settings screen, reusing ticket 2's
// account machinery (createAccountRow, validateCredentials, createError) rather
// than a second create path.
func (s *server) inviteAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := r.FormValue("role")
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
			section: "accounts", acctError: msg, acctUsername: username, acctRole: role,
		})
	}

	if role != roleAdmin && role != roleViewer {
		fail("Role must be admin or viewer.")
		return
	}
	if msg := validateCredentials(username, password); msg != "" {
		fail(msg)
		return
	}
	if _, err := s.createAccountRow(r, username, role, password); err != nil {
		fail(createError(err))
		return
	}
	http.Redirect(w, r, "/settings?tab=access", http.StatusSeeOther)
}

// setAccountRole reassigns an account's role. It refuses to demote the last
// admin: an operator must never be able to strip the final admin and lock every
// remaining account out of every mutation.
func (s *server) setAccountRole(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{section: "accounts", roleError: msg})
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
	http.Redirect(w, r, "/settings?tab=access", http.StatusSeeOther)
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
	case "sources":
		err = s.fillSourcesSection(r, data)
	case "channels":
		err = s.fillChannelsSection(r, f, data)
	case "integrations":
		err = s.fillIntegrationsSection(r, data)
	case "messages":
		err = s.fillMessagesSection(r, data)
	case "delivery":
		err = s.fillDeliverySection(r, f, data)
	case "access":
		err = s.fillAccessSection(r, acct, f, data)
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
// register (ADR-0039/ADR-0081), the two retention dials, and the verge-core hot
// port set (§3.5). verge-core folds here as the delivery-oriented dial screen it
// most resembles, keeping its frequency edit and read-only sensitive half intact.
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

	// verge-core composition.
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

// fillAccessSection carries the account surface relocated from the temporary
// /account home (#277): the account list, invite, role controls and TOTP status,
// plus the SSO honest empty state. A viewer sees only their own account.
func (s *server) fillAccessSection(r *http.Request, acct db.Account, f settingsForms, data map[string]any) error {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil {
		return err
	}
	data["Accounts"] = toAccountRows(accounts, acct.ID)
	data["AcctError"] = f.acctError
	data["AcctUsername"] = f.acctUsername
	data["AcctRole"] = f.acctRole
	data["RoleError"] = f.roleError
	return nil
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
func validateChannelURL(raw string) (string, string) {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return "", "Enter an absolute URL, like https://example.com/hook."
	}
	switch u.Scheme {
	case "https":
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
