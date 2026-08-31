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
	ID       int64
	URL      string
	Drift    bool
	Coverage bool
	Clock    bool
	// Classes is the store vocabulary this channel carries, in vocabulary order —
	// the display tags (#26f, never a hardcoded set in the tmpl). ClassStates is the
	// full vocabulary with each class's checked flag for the edit-disclosure form.
	Classes     []string
	ClassStates []classState
	Enabled     bool
	HasSecret   bool
	By          string
	At          string
}

// classState is one routing class in the channel vocabulary with its checked flag —
// the shape both the create form's .ClassOptions and a channel's per-row .ClassStates
// render from, so the class checkboxes/badges come from the store's vocabulary rather
// than a hardcoded set (#26f).
type classState struct {
	Name    string
	Checked bool
}

// channelClasses is the store's routing-class vocabulary (the route_drift /
// route_coverage / route_clock columns, ADR-0091). The tmpl renders class
// checkboxes and badges from this, never from a literal set baked into the markup.
var channelClasses = []string{"drift", "coverage", "clock"}

// accountRow is one account in the management list. It carries no password hash
// and no TOTP secret — managing an account needs neither.
type accountRow struct {
	ID          int64
	Username    string
	Initials    string
	Role        string
	TotpEnabled bool
	At          string
	IsSelf      bool
}

// sessionRow is one active session in the admin-wide sessions view (#407). It
// carries no token hash — the listing query never projects the secret — only whose
// session it is (Account) and at what Role, the device derived from the stored
// User-Agent, the source IP, and the relative last-active reading. Current marks the
// one row whose cookie is making this request (Settings.jsx's `current`): it wears the
// "(you)" marker and shows no revoke control, so the operator can never sign their own
// browser out from the admin-wide surface.
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

// retentionView renders the three dials and who last moved them.
type retentionView struct {
	ObservationCurrencyDays int64
	DispatchCadenceMultiple int64
	TranscriptCurrencyDays  int64
	UpdatedBy               string
	UpdatedAt               string
}

// addressCapView renders the address-scope cap control (#888 / Settings #206,
// ADR-0127, Variant C — the policy-forward dial). It front-loads the cost of a raised
// cap at policy time so the declaration can stay a flat within-policy confirm: the
// largest scope the cap admits (as a prefix per family), the dispatch load a cap-sized
// scope puts on each enabled address-scope scan per cadence, and the projected
// evidential-observation disk growth (scaling ADR-0041's /22 ≈ 13 GB/year grounding).
// The operator chooses the lag they accept HERE, when they set the cap, not when they
// declare. Nothing here branches on the number; it is a readout, never a gate.
type addressCapView struct {
	Cap            int64
	CapLabel       string // the cap as a comma-grouped count, e.g. "262,144"
	LargestScopeV4 string // the widest IPv4 prefix the cap admits, e.g. "/14"
	LargestScopeV6 string // the widest IPv6 prefix the cap admits, e.g. "/110"
	DiskProjection string // projected evidential growth, e.g. "≈ 3.3 TB / year"
	SweepLoad      []capSweepLine
	UpdatedBy      string
	UpdatedAt      string
}

// capSweepLine is one enabled address-scope scan's dispatch load at the current cap: a
// cap-sized scope is Probes one-address Batches (ADR-0005) dispatched every Cadence.
// The Probes figure is pure arithmetic over the cap and the scan's own cadence_seconds —
// no throughput is assumed, so it promises no completion the model does not (ADR-0127).
//
// Effective is the separate predicted-vs-effective cadence #891 states on the Scans
// surface (#847): the worst-case time one full cap-sized pass takes at the estate
// packet ceiling — declared size x probed ports x attempts / rate, the same figures
// ADR-0047 prices with. It is arithmetic, not a new domain term, and it never appears
// on Coverage (Coverage is evidential; Scans is operational). Cadence is the predicted
// cadence; Effective is the effective cadence; Outpaces is true when one pass cannot
// finish inside the predicted cadence, so the trailing edge lags (#847, reported on
// Coverage as the honest shortfall, never hidden). ADR-0005's skip events confirm the
// prediction in operation.
type capSweepLine struct {
	Scan      string // the scan kind, e.g. "hot"
	Cadence   string // its declared (predicted) cadence label, e.g. "daily"
	Probes    string // the cap as a comma-grouped probe count, e.g. "262,144"
	Effective string // the effective cadence — one full pass, e.g. "≈ 6 days"; "" when unknown
	Outpaces  bool   // true when Effective exceeds the predicted Cadence (the honest lag)
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
	// Latency is the measured connect round-trip label ("34ms") or empty when
	// unmeasured; Unverified marks a vantage that makes no exposure claims until
	// re-verified (its availability reads "unverified"). The spec VantageCard renders
	// the dashed border and no-claims note off Unverified (#26c).
	Latency    string
	Unverified bool
	Avail      string
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

	retError      string
	retObs        string
	retDispatch   string
	retTranscript string

	// address-scope cap (#888, ADR-0127). capError is the inline error on the cap
	// control (Settings · Scans); capValue echoes a rejected value back so the operator
	// does not retype it.
	capError string
	capValue string

	vcError string
	vcPort  string

	// sources (#26). sourceError is an inline error on the sources tab (a bad id or a
	// rejected enable), echoed above the tier cards.
	sourceError string

	// cold + probers (#21d): the full-range opt-in and prober provisioning acts
	// relocated from /scope. coldError is an inline error on the Scans tab's cold-tier
	// region; the prober fields echo a rejected provision form back on the Vantages tab.
	coldError   string
	proberError string
	proberHost  string
	proberPort  string
	proberUser  string

	// sessions (#407). revokeAccountID/revokeAccountError re-open the typed-name
	// revoke-all-for-account ConfirmDialog on a mismatch, exactly as the Team
	// remove-account dialog re-opens through removeID/removeError.
	revokeAccountID    int64
	revokeAccountError string

	// restore (#391/B4, ADR-0124): the Instance tab's Restore card state. restoreError
	// is the inline failure line (a fixed message keyed to a redirect code, never
	// reflected text). preflight surfaces a staged, validated archive ready to apply;
	// restoreConfirm re-shows that same staged archive as the typed-confirm dialog when
	// ?restore-confirm=1. All nil/empty on a normal Instance render.
	restoreError   string
	preflight      *restorePreflightView
	restoreConfirm *restoreConfirmView
}

// settingsTabs is the sub-tab order of the Settings screen, ported from
// examples/console/Settings.jsx's SettingsNav groups: Scanning (scans, vantages),
// Access (single sign-on, team, audit log), Discovery (sources, port aperture),
// Instance (health), then Delivery (channels, integrations, messages, delivery
// record). Each is reached at /settings?tab=<id>.
var settingsTabs = []string{
	"scans", "vantages",
	"sso", "team", "audit", "api", "sessions",
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
	case "scans", "cold":
		return "scans"
	default:
		return "scans"
	}
}

func (s *server) settingsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// A VERGE_DEV build serves the design's curated fixtures.json settings slice so each
	// section renders byte-for-byte for the pixel-parity harness (the 19 golden states).
	// It touches no table — the twin of the render-goldens settings case, both stamping
	// the same "settings" holes from the same fixture. A real deployment renders the live
	// projection below (renderSettings). The viewer's forbidden state never reaches here:
	// requireSettingsAdmin refuses it first (settingsForbidden, the error-page).
	if s.devMode {
		s.render(w, r, "settings", s.settingsFixtureData(acct, r))
		return
	}
	q := r.URL.Query()
	forms := settingsForms{
		tab:    validTab(q.Get("tab")),
		notice: sessionsNotice(q.Get("notice")),
	}
	// Restore card state (#391/B4, ADR-0124) rides the Instance tab only. A failed
	// pre-flight or apply redirects here with ?restore-error=<code>, mapped to a fixed
	// line (never reflected text). A completed pre-flight left the validated archive
	// staged for this admin; surface it as the warn callout, or — with ?restore-confirm=1
	// — as the typed-confirm dialog.
	if forms.tab == "instance" {
		forms.restoreError = restoreErrorMessage(q.Get("restore-error"))
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

// sessionsNotice maps a redirect's notice code to a fixed success line so a
// completed admin session act (#407) confirms on the reloaded surface without
// reflecting arbitrary query text into the page. An unknown code renders no notice.
func sessionsNotice(code string) string {
	switch code {
	case "revoked":
		return "Session revoked — its next request lands on the sign-in screen."
	case "revoked-account":
		return "Every session for that account was revoked."
	default:
		return ""
	}
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
	log.Printf("web: invite minted at role %q; accept it at %s (expires in %s)", role, link, inviteTTL) // #nosec G706 (role is enum-validated admin|viewer; link is server-constructed)
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

// updateRetention persists the three dial values. The observation and dispatch
// floors are DERIVED not asserted (ADR-0094) — never presented as an operator
// choice. The observation-currency dial (#208, §4.6) floors at the tightest
// observation bound in force: k cadences of the tightest enabled Scan, below which
// the control changes no row at all. The Dispatch dial (#209, §4.6) floors at k
// cadences of the slowest enabled Scan. The transcript-currency dial (#868,
// raw-job-output spec §4, ADR-0126) is a whole number of days: 0 is the explicit
// unbounded opt-out, and a positive value is floored UP to 1 day by the retirer
// (retention.TranscriptFloorDays), so no positive whole-day value is rejected here.
// For all three, 0 is always allowed. Deletion of expired rows is a structurally
// separate path (internal/retention), never reached from here.
func (s *server) updateRetention(w http.ResponseWriter, r *http.Request, acct db.Account) {
	obsRaw := strings.TrimSpace(r.FormValue("observation_currency_days"))
	dispRaw := strings.TrimSpace(r.FormValue("dispatch_cadence_multiple"))
	transRaw := strings.TrimSpace(r.FormValue("transcript_currency_days"))
	fail := func(msg string) {
		s.renderSettings(w, r, acct, settingsForms{
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
	trans, err := strconv.ParseInt(transRaw, 10, 64)
	if err != nil || trans < 0 {
		fail("Transcript retention must be a whole number of days, zero or more.")
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
		TranscriptCurrencyDays: trans,
		UpdatedBy:              pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update retention", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=delivery", http.StatusSeeOther)
}

// updateAddressCap sets the operator address-scope cap (#888 / Settings #206,
// ADR-0127). The cap has NO upper bound — ADR-0127 removes the ceiling above the
// operator value, so the only friction is the deliberate act of raising it; the sole
// guard here is a whole number of addresses, one or more. It persists on the
// instance_config singleton and is read at declaration (server.addressCap), so a raise
// takes effect on the next declaration and a lower value never invalidates a scope
// declared under a higher cap. A rejected value re-renders the Scans tab with the
// echo state and a 400, exactly as the retention dials do.
func (s *server) updateAddressCap(w http.ResponseWriter, r *http.Request, acct db.Account) {
	raw := strings.TrimSpace(r.FormValue("address_cap"))
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 1 {
		s.renderSettings(w, r, acct, settingsForms{
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
	http.Redirect(w, r, "/settings?tab=scans", http.StatusSeeOther)
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
		"NavActive": "settings", "Tab": active, "DesignTokens": true,
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

	status := http.StatusOK
	if f.section != "" {
		status = http.StatusBadRequest
	}
	s.renderStatus(w, r, status, "settings", data)
}

// fillVantagesSection lists the provisioned measurement positions (CONTEXT.md
// "Vantage"). A read-only display: provisioning lives on Scope, and a vantage is
// never a probe/scanner/agent here.
func (s *server) fillVantagesSection(r *http.Request, f settingsForms, data map[string]any) error {
	rows, err := s.store.ListVantages(r.Context())
	if err != nil {
		return err
	}
	// The prober provisioning form's echo (#21d, relocated from /scope): a rejected
	// provision re-renders the Vantages tab with its own error and typed values.
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
	// The create form's class checkboxes render from the store vocabulary (#26f),
	// pre-checked to the create-form defaults (all three, or the operator's echoed
	// selection after a rejected create).
	defaults := map[string]bool{"drift": chDrift, "coverage": chCoverage, "clock": chClock}
	opts := make([]classState, 0, len(channelClasses))
	for _, name := range channelClasses {
		opts = append(opts, classState{Name: name, Checked: defaults[name]})
	}
	data["ClassOptions"] = opts
	return nil
}

// initialsFromUsername derives a member's initial-avatar label from their username
// — the first two letters of the local part, uppercased (Settings.jsx derives the
// avatar from the username, no new datum). A single-character local part yields one
// letter; an empty username yields the empty string.
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
	data["RetTranscript"] = f.retTranscript
	return nil
}

// fillAPISection carries the read-only /api/v1 opt-in surface (#390, ADR-0123 pending
// A1). It reads the single instance_config row: .API{Enabled,By,At}, where By/At are the
// dated act of the CURRENT state (who last flipped the surface on, when) and both stay
// nil while it has never been enabled. Enabling is admin-only (the toggle is behind
// .IsAdmin in the tmpl, its POST /settings/api handler is A4); a viewer reaches this one
// Settings tab read-only (requireSettingsAdmin lets ?tab=api through) and sees the state
// and note without a button. The bearer verification, the enable POST and the live
// enabled render are the A-cluster's; this lands the disabled/read baseline.
func (s *server) fillAPISection(r *http.Request, data map[string]any) error {
	cfg, err := s.store.GetInstanceConfig(r.Context())
	if err != nil {
		return err
	}
	api := map[string]any{"Enabled": cfg.ApiEnabled}
	if cfg.ApiEnabled {
		// By/At describe the current enabled state — resolve the author username by the
		// same join toRetentionView uses (settings.go), and format the instant in UTC.
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
		editByPort[uint16(e.Port)] = e.Action                                                  // #nosec G115 (DB port written only via 1..65535-validated edit path)
		edits = append(edits, vergecore.FrequencyEdit{Port: uint16(e.Port), Action: e.Action}) // #nosec G115 (DB port written only via 1..65535-validated edit path)
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

// fillSessionsSection carries the admin-wide sessions surface (#407, ADR-0117): every
// account's live browser sessions across the deployment, joined to the owning account's
// username and role, grouped by account then recency (the query's own order). The
// listing never projects the token hash. It also opens the two ConfirmDialogs by query
// param — single-session revoke (?revoke=<sessionID>) and revoke-all-for-account
// (?revoke-account=<accountID>, typed-name), the latter re-opened by a rejected POST
// through settingsForms — matching the Team surface's dialog idiom.
func (s *server) fillSessionsSection(r *http.Request, f settingsForms, data map[string]any) error {
	now := s.now()
	rows, err := s.store.ListAllActiveSessions(r.Context(), pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return err
	}
	// The row whose cookie is making this request is marked current (Settings.jsx's
	// `current`) — the same resolution the Profile sessions surface uses (auth.go's
	// currentSessionID). ok=false when no cookie resolves, in which case no row is
	// treated as current.
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
	// Single-revoke ConfirmDialog: opened by ?revoke=<sessionID>. The dialog reads the
	// session's own details from the already-gathered list.
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
	// Revoke-all-for-account typed-name dialog: opened by ?revoke-account=<accountID> (GET)
	// or re-opened by a rejected typed-name POST through f.revokeAccountID. The username the
	// operator must type is taken from any of that account's sessions in the list.
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

// revokeSessionAdmin revokes any single session by id — the admin-wide kill of one
// active session (#407). It is admin-gated (requireAdmin) and NOT owner-scoped
// (RevokeSessionByIDForAdmin), reached only through the per-row ConfirmDialog. It is
// idempotent: an unparseable or already-revoked id redirects back cleanly. The very
// next request carrying that session's cookie resolves no live session and is bounced to
// /login (#405).
func (s *server) revokeSessionAdmin(w http.ResponseWriter, r *http.Request, _ db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/settings?tab=sessions", http.StatusSeeOther)
		return
	}
	if err := s.store.RevokeSessionByIDForAdmin(r.Context(), db.RevokeSessionByIDForAdminParams{
		ID: id, RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "admin revoke session", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=sessions&notice=revoked", http.StatusSeeOther)
}

// revokeAccountSessions revokes every live session for one account — the offboarding
// kill (#407). It passes through a typed-name gate exactly as the Team remove-account
// act does: the operator must type the account's username to confirm, and it is reached
// only through the revoke-all ConfirmDialog. It never touches the account's membership,
// role or personal tokens — only its live sessions — and is idempotent.
func (s *server) revokeAccountSessions(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("account_id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/settings?tab=sessions", http.StatusSeeOther)
		return
	}
	target, err := s.store.GetAccountByID(r.Context(), id)
	if err != nil {
		// The account is gone; there is nothing to revoke and no dialog to re-open.
		http.Redirect(w, r, "/settings?tab=sessions", http.StatusSeeOther)
		return
	}
	if strings.TrimSpace(r.FormValue("confirm_name")) != target.Username {
		s.renderSettings(w, r, acct, settingsForms{
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
	http.Redirect(w, r, "/settings?tab=sessions&notice=revoked-account", http.StatusSeeOther)
}

// sessionDeviceFromUA describes a session from a stored User-Agent string — a real
// derivation of what the client sent, never a fabricated device. It is the string-typed
// twin of auth.go's sessionDevice (which reads the live request); the admin surface only
// holds the persisted UA, so it derives the same label from that. An absent or
// unrecognised agent degrades to a plain label rather than a guess.
func sessionDeviceFromUA(ua string) string {
	if ua == "" {
		return "Unknown device"
	}
	// A non-browser client (the verge CLI / API automation) announces itself as verge-cli
	// and carries its user@host in the parenthetical, e.g. "verge-cli/1.0 (verge@build-07)".
	// Label such a session "CLI · <host>" so it reads distinctly from a browser device
	// (Profile sessions, "CLI · verge@build-07"). A verge-cli client with no parenthetical
	// falls back to a bare "CLI".
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

// fillInstanceSection carries the instance-health tab (Settings.jsx InstanceSection)
// as real reads only — no fabricated version string, uptime figure, or queue depth
// where the datum does not exist. What is real: the licence/build stance (AGPL-3.0,
// self-hosted), the process uptime since start, the build version, the applied-vs-embedded
// migrations count, that Postgres answered this render, the provisioned vantage fleet with
// each vantage's availability, and the release check's opt-in flag + cached last result
// (#391, ADR-0124). The Backup/Restore card bodies are the B3/B4 clusters'.
func (s *server) fillInstanceSection(r *http.Request, f settingsForms, data map[string]any) error {
	ctx := r.Context()
	// Real host facts only (#26h): the licence stance, the process uptime since start, and
	// the build version off VERGE_VERSION (buildVersion, the same the auth footer reads).
	// Queue depth, disk and Postgres are wired from live reads below (#633,
	// WORK-ORDER-DOGFOOD-R1 item 3), each best-effort: a failed read leaves its hole empty
	// and the figure collapses, never a guessed number.
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

	// Queue depth — the real count of in-flight queue jobs (ready + running) across the
	// recent dispatches: the work waiting on the queue, the "subjects waiting" figure.
	if rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit); err == nil {
		var waiting int64
		for _, row := range rows {
			waiting += row.Ready + row.Running
		}
		inst["QueueDepth"] = waiting
	} else {
		log.Printf("web: instance: queue depth: %v", err)
	}

	// Disk — a real Statfs of the working-directory volume on the deployment host
	// (diskstat_unix.go). Off unix (dev on Windows) diskUsage reports ok=false and the
	// figure collapses rather than fabricate one.
	if used, total, ok := diskUsage("."); ok {
		inst["DiskDetail"] = diskLabel(used, total)
		inst["DiskPct"] = int(used * 100 / total) // #nosec G115 -- used<=total (guarded in diskUsage), so the percentage is 0..100
	}

	// Database — real pg_database_size and server version off the running Postgres.
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

	// Migrations — best-effort applied-vs-embedded count (#391). max(version_id) in
	// goose's ledger against the versions embedded in the binary (db/migrations); a
	// version embedded but not yet applied is pending. A read that fails (nil pool off
	// the pixel harness, or a query error) leaves the badge absent rather than guessing —
	// the tmpl's {{with .Migrations}} collapses, never a fabricated "schema current".
	if pending, ok := s.migrationsPending(ctx); ok {
		inst["Migrations"] = map[string]any{"Pending": pending}
	}

	// Release — the Version & updates card (#391, ADR-0124: check + surface + guide,
	// never self-replace). The single instance_config row carries the opt-in flag and the
	// worker's cached last check (B5 writes it). State is disabled when the check is opted
	// out, else newer/current from the cache. The host steps are release-authored and
	// literal — the UI never composes a shell — so they ride a fixed constant, not a
	// derivation. A failed config read leaves the release block absent.
	if cfg, err := s.store.GetInstanceConfig(ctx); err == nil {
		release := map[string]any{
			"CheckEnabled": cfg.UpdateCheckEnabled,
			"Steps":        updateHostSteps,
		}
		state := "disabled"
		if cfg.UpdateCheckEnabled {
			// Enabled ⇒ newer or current (disabled means the check is off). A "newer"
			// carries the cached latest version + notes; anything else reads as current
			// — nothing newer known — never a guess at an unseen release.
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

		// Backup card (#391, ADR-0124, B3). A synchronous streamed backup never sets
		// InProgress/Streamed/SizeHint/Percent, so those stay unset and the tmpl's
		// {{if .InProgress}} branch collapses to the download button + last-backup note.
		// .Backup itself must be non-nil for {{with .Backup}} to render the button at all;
		// the record is null (empty LastAt) until the first UI backup, when SetLastBackup
		// (cmd/web/backup.go) stamps it. LastAt mirrors the Release CheckedAt format.
		backup := map[string]any{"LastAt": "", "LastSize": ""}
		if cfg.LastBackupAt.Valid {
			backup["LastAt"] = cfg.LastBackupAt.Time.UTC().Format("2006-01-02 15:04 UTC")
			backup["LastSize"] = humanBytes(cfg.LastBackupSize.Int64)
		}
		inst["Backup"] = backup
	} else {
		log.Printf("web: instance: release config: %v", err)
	}

	// Restore card state (#391/B4, ADR-0124). All three holes collapse when unset: a
	// plain Instance render carries no staged pre-flight and no error, so the card shows
	// its "Choose archive…" upload form. A completed pre-flight surfaces .Preflight (the
	// warn callout), ?restore-confirm=1 surfaces .RestoreConfirm (the typed-confirm
	// dialog), and any refusal surfaces .RestoreError. The three are mutually exclusive by
	// construction of the handlers that set them.
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

// updateHostSteps are the literal, release-authored host commands the Version & updates
// card prints when a newer release is available (#391, ADR-0124). Verge never rewrites
// its own image — the swap is a host action — so the UI composes no shell: it renders
// these exact lines. Until B5 threads a feed-delivered list through the release cache
// they live here as the shipped constant, matching the design fixture line-for-line.
var updateHostSteps = []string{
	"# on the host — verge cannot rewrite its own image",
	"docker compose pull",
	"docker compose up -d web worker",
	"docker compose exec web verge migrate status",
}

// migrationsPending is the best-effort applied-vs-embedded migrations count the
// Version & updates badge shows (#391): how many embedded goose migrations carry a
// version newer than the highest goose has applied. It reads goose's ledger with a raw
// pool query (not sqlc — internal/db stays untouched this round) and the embedded set
// from the same migrations.FS the binary applies at boot. Best-effort: a nil pool (off
// the pixel harness) or any read error returns ok=false, so the badge collapses rather
// than fabricate a "schema current".
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

// migrationVersion parses the leading integer of a goose migration filename
// (e.g. "23000_instance_config.sql" → 23000), the version goose records in its ledger.
// A name with no leading integer is not a numbered migration and reports ok=false.
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

// updateCheckToggle opts the worker's daily release-feed check in or out (#391, ADR-0124).
// The hidden `enabled` field carries the flip target the Version & updates toggle computed
// from the current state; SetUpdateCheckEnabled stamps who acted and when. While off the
// worker never dispatches a check — air-gap-safe (B5 honours the flag). Admin-gated
// (requireAdmin) with a PRG back to the Instance tab so a reload does not re-post.
func (s *server) updateCheckToggle(w http.ResponseWriter, r *http.Request, acct db.Account) {
	enabled := r.FormValue("enabled") == "true"
	if err := s.store.SetUpdateCheckEnabled(r.Context(), db.SetUpdateCheckEnabledParams{
		UpdateCheckEnabled:   enabled,
		UpdateCheckUpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "set update check enabled", err)
		return
	}
	http.Redirect(w, r, "/settings?tab=instance", http.StatusSeeOther)
}

// apiToggle flips the read-only /api/v1 surface on or off (#390, ADR-0123). The hidden
// `enabled` field carries the flip target the API access toggle computed from the current
// state; SetAPIEnabled stamps who acted and when, so the card renders the dated act of the
// current state (Enabled by … · …). Enabling makes every minted personal token answer
// GET /api/v1/… with its account's read access — read-only, always, no write surface to
// enable; disabling returns /api/v1 to answering nothing on every path (surface off beats
// auth) and leaves every token inert. Admin-gated (requireAdmin); a PRG back to the API tab
// so a reload does not re-post, riding the shell toast pipeline the way other acts do.
func (s *server) apiToggle(w http.ResponseWriter, r *http.Request, acct db.Account) {
	enabled := r.FormValue("enabled") == "true"
	if err := s.store.SetAPIEnabled(r.Context(), db.SetAPIEnabledParams{
		ApiEnabled:   enabled,
		ApiUpdatedBy: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "set api enabled", err)
		return
	}
	if enabled {
		s.toastRedirect(w, r, "/settings?tab=api", "ok", "API access enabled",
			"Personal tokens now answer GET /api/v1/… — read-only, always.")
		return
	}
	s.toastRedirect(w, r, "/settings?tab=api", "neutral", "API access disabled", "")
}

// diskLabel renders the used / total volume figure the instance-health disk row shows
// (e.g. "24.8 / 40 GB", the fixture format): both in gibibytes, used carrying one
// decimal and total rounded, the unit named once. The percentage rides the bar
// (.DiskPct) separately, so it is not repeated here.
func diskLabel(used, total uint64) string {
	const gb = 1 << 30
	return fmt.Sprintf("%.1f / %.0f GB", float64(used)/gb, float64(total)/gb)
}

// humanBytes renders a byte count as the terse GB/MB/KB figure the database-size row
// shows (e.g. "4.2 GB"). It picks the largest unit that keeps the number readable, so a
// fresh database reads in MB rather than a long fraction of a GB.
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

// pgLabel renders the "postgres <major>" label from a raw server_version string (e.g.
// "16.4" or "16.4 (Debian 16.4-1)" → "postgres 16"). A version with no leading integer
// falls back to a bare "postgres" rather than a malformed label.
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

// addressScopeScanKinds are the scan tiers that sweep an address scope by enumerating
// it one address per Batch (ADR-0127: the hot/cold fan-out). The bounded tiers (dns,
// zone, tls-acceptance, http-identity, ct) enumerate over name scopes or the resolved
// service population, not an address-scope sweep, so a raised cap puts no per-address
// dispatch load on them — they are absent from the cap control's sweep-load readout.
var addressScopeScanKinds = map[string]bool{"hot": true, "cold": true}

// toAddressCapView builds the policy-forward cap dial (#888, ADR-0127 Variant C) from
// the persisted cap, the enabled scans and the account list (to name who last set it).
// It reads the effective cap through the same DefaultAddressCap fallback the
// declaration path uses (server.addressCap), so the readout matches what a declaration
// would actually check.
func toAddressCapView(cfg db.GetInstanceConfigRow, scans []db.Scan, accounts []db.ListAccountsRow) addressCapView {
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
		// The effective cadence (#891, decision #847): one full cap-sized pass at the
		// estate packet ceiling. A scan with no probed ports (unknown kind) yields no
		// figure. Outpaces compares the worst-case pass to the declared cadence — a pass
		// longer than its cadence cannot finish in time, and the trailing edge lags.
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

// projectedEvidentialDiskPerYear projects the evidential-observation disk growth a
// cap-sized address scope drives per year, scaling ADR-0041's grounding: one declared
// /22 (1024 addresses) grows the evidential corpus ~13 GB/year, and rows are linear in
// the address count, so the projection is 13 GB/year × cap/1024. It is a projection,
// stated as one (the "≈"), not a measurement; ADR-0041 moved this figure onto the
// policy dial so the operator prices the disk cost beside the cadence lag before they
// raise the cap.
func projectedEvidentialDiskPerYear(addrCap int64) string {
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

// addressScopePorts is the probed-port count the effective-cadence projection (#891)
// multiplies the address count by, per address-scope scan tier. hot probes verge-core's
// TCP pairs — 131 on default settings (internal/vergecore) — and cold connects to every
// TCP port, 1-65535. These are the shipped nominal figures ADR-0047 prices with; an
// operator's frequency edits shift hot's count, so the projection states "≈", exactly as
// projectedEvidentialDiskPerYear projects off a fixed grounding rather than the live set.
var addressScopePorts = map[string]int64{"hot": 131, "cold": 65535}

// effectiveCadenceSeconds is the worst-case time one full cap-sized address-scope pass
// takes at the estate packet ceiling (#891, decision #847): (addresses x ports x attempts)
// / rate. attempts is 1 + the connect-outcome retry budget — the pass where every probe
// exhausts its retries — so the figure never understates the lag (ADR-0127 promises no
// completion the model cannot keep). rate and retries read from the leaf's declared safety
// profile so a change to either moves this figure with it. The math is float64: the cap has
// no ceiling (ADR-0127), so a very large cap would overflow int64, and this is a stated
// projection ("≈"), not an exact instant.
func effectiveCadenceSeconds(addresses, portsPerAddress int64) float64 {
	p := connectoutcome.DefaultProfile()
	rate := float64(p.GlobalPacketsPerSec)
	if rate <= 0 {
		rate = 1
	}
	attempts := float64(1 + p.Retries)
	return float64(addresses) * float64(portsPerAddress) * attempts / rate
}

// projectedPassLabel humanizes an effective-cadence projection in seconds as one coarse
// "≈" figure — the readout compares against the predicted cadence word, so a single unit
// reads cleaner than a two-unit countdown. It rounds to the largest fitting unit, from
// minutes up to years, and never renders a bare zero.
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
