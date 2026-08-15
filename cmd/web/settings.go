package main

import (
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Settings screen is the operator's dials (v1 spec §6.1): accounts, Channels
// and the two retention dials. Every mutation it hosts is an authenticated admin
// act (§4.3, notification-channels.md §2) — the whole destination is reached only
// through requireAdmin, so viewing it and mutating from it are both admin-gated.

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

// settingsForms carries the echo state of the Settings screen's forms so a
// rejected submission on one section leaves its own error and typed values in
// place without disturbing the others. section names the section that failed and
// drives the response status.
type settingsForms struct {
	section string // "", "accounts", "channels" or "retention"

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
}

func (s *server) settingsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.renderSettings(w, r, acct, settingsForms{})
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
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// --- retention -------------------------------------------------------------

// updateRetention persists the two dial values. Both floor at zero for now
// (§4.6): the real floors — the tightest observation bound in force, and k
// cadences of the slowest enabled Scan for Dispatch — are validated once tickets
// 28/29 define them. Until then zero means no operator floor.
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
	if err := s.store.UpdateRetentionSettings(r.Context(), db.UpdateRetentionSettingsParams{
		ObservationCurrencyDays: obs, DispatchCadenceMultiple: disp,
		UpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update retention", err)
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// --- render ----------------------------------------------------------------

func (s *server) renderSettings(w http.ResponseWriter, r *http.Request, acct db.Account, f settingsForms) {
	accounts, err := s.store.ListAccounts(r.Context())
	if err != nil {
		s.serverError(w, "list accounts", err)
		return
	}
	channels, err := s.store.ListChannels(r.Context())
	if err != nil {
		s.serverError(w, "list channels", err)
		return
	}
	ret, err := s.store.GetRetentionSettings(r.Context())
	if err != nil {
		s.serverError(w, "get retention", err)
		return
	}

	// The create-channel form defaults to all three classes; only a rejected
	// create echoes the operator's own selection back.
	chDrift, chCoverage, chClock := true, true, true
	if f.section == "channels" {
		chDrift, chCoverage, chClock = f.chanDrift, f.chanCoverage, f.chanClock
	}

	status := http.StatusOK
	if f.section != "" {
		status = http.StatusBadRequest
	}
	s.renderStatus(w, status, "settings", map[string]any{
		"Title": "Settings", "Account": acct, "IsAdmin": true,
		"Accounts":  toAccountRows(accounts, acct.ID),
		"Channels":  toChannelViews(channels),
		"Retention": toRetentionView(ret, accounts),
		"AcctError": f.acctError, "AcctUsername": f.acctUsername, "AcctRole": f.acctRole,
		"RoleError": f.roleError,
		"ChanError": f.chanError, "ChanURL": f.chanURL,
		"ChanDrift": chDrift, "ChanCoverage": chCoverage, "ChanClock": chClock,
		"RetError": f.retError, "RetObs": f.retObs, "RetDispatch": f.retDispatch,
	})
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
