package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/qr"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

const (
	sessionCookie = "verge_session"
	pendingCookie = "verge_totp_pending"
	issuer        = "Verge ASM"
)

// dummyHash is a valid bcrypt hash compared against when a login names an
// account that does not exist, so a missing username and a wrong password take
// the same time and neither is distinguishable from the outside.
var dummyHash, _ = auth.HashPassword("verge-timing-equaliser")

// authedHandler is a handler that has already resolved the caller's account.
type authedHandler func(w http.ResponseWriter, r *http.Request, acct db.Account)

// currentAccount resolves the signed session cookie to an account. Identity
// comes only from the cookie — never from a proxy-supplied header — so no
// reverse-proxy forward-auth path exists to trust (v1 spec §4.3, §7). The role
// is read live from the row, so a deleted account fails here at once.
func (s *server) currentAccount(r *http.Request) (db.Account, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return db.Account{}, false
	}
	sess, err := auth.VerifySession(s.key, c.Value, auth.KindSession, s.now())
	if err != nil {
		return db.Account{}, false
	}
	acct, err := s.store.GetAccountByID(r.Context(), sess.AccountID)
	if err != nil {
		return db.Account{}, false
	}
	return acct, true
}

func (s *server) requireLogin(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acct, ok := s.currentAccount(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		h(w, r, acct)
	}
}

func (s *server) requireAdmin(h authedHandler) http.HandlerFunc {
	return s.requireLogin(func(w http.ResponseWriter, r *http.Request, acct db.Account) {
		if acct.Role != roleAdmin {
			s.forbidden(w, r)
			return
		}
		h(w, r, acct)
	})
}

const (
	roleAdmin  = "admin"
	roleViewer = "viewer"
)

// --- setup -----------------------------------------------------------------

func (s *server) setupForm(w http.ResponseWriter, r *http.Request) {
	if s.setupClosed(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.render(w, "setup", map[string]any{"Title": "Setup", "Token": r.URL.Query().Get("token")})
}

func (s *server) setupSubmit(w http.ResponseWriter, r *http.Request) {
	// Serialise the closed-check and the create so two valid first-boot POSTs
	// cannot both observe zero accounts and each create an admin.
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if s.setupClosed(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	token := r.FormValue("token")
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if !auth.TokensEqual(token, s.setupToken) {
		s.render(w, "setup", map[string]any{"Title": "Setup", "Token": token, "Error": "Invalid setup token."})
		return
	}
	if msg := validateCredentials(username, password); msg != "" {
		s.render(w, "setup", map[string]any{"Title": "Setup", "Token": token, "Error": msg})
		return
	}
	if _, err := s.createAccountRow(r, username, roleAdmin, password); err != nil {
		s.render(w, "setup", map[string]any{"Title": "Setup", "Token": token, "Error": createError(err)})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// setupClosed reports whether the bootstrap window is shut: it is shut once any
// account exists, which is what makes the setup token single-use — the first
// admin it creates closes it.
func (s *server) setupClosed(r *http.Request) bool {
	if s.setupToken == "" {
		return true
	}
	n, err := s.store.CountAccounts(r.Context())
	if err != nil {
		log.Printf("web: setup: count accounts: %v", err)
		return true
	}
	return n > 0
}

// --- login -----------------------------------------------------------------

func (s *server) loginForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentAccount(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	data := map[string]any{"Title": "Sign in", "SSOProviders": s.enabledSSOProviders(r.Context())}
	// A freshly accepted invite lands here (invite acceptance creates the account
	// but grants no session — the new operator signs in with the credentials they
	// just set), so surface a notice rather than a bare form.
	if r.URL.Query().Get("invited") != "" {
		data["Notice"] = "Account created. Sign in with your new credentials."
	}
	s.render(w, "login", data)
}

func (s *server) loginSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		auth.CheckPassword(dummyHash, password) // equalise timing with the found path
		s.render(w, "login", s.loginData(r.Context(), "Invalid username or password."))
		return
	}
	if !auth.CheckPassword(acct.PasswordHash, password) {
		s.render(w, "login", s.loginData(r.Context(), "Invalid username or password."))
		return
	}
	if acct.TotpEnabled {
		if !s.setSignedCookie(w, r, pendingCookie, auth.KindPending, acct.ID, s.pendingTTL) {
			return
		}
		s.render(w, "totp", map[string]any{"Title": "Two-factor"})
		return
	}
	s.completeLogin(w, r, acct.ID)
}

func (s *server) loginTOTP(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(pendingCookie)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sess, err := auth.VerifySession(s.key, c.Value, auth.KindPending, s.now())
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	acct, err := s.store.GetAccountByID(r.Context(), sess.AccountID)
	if err != nil || !acct.TotpEnabled || !acct.TotpSecret.Valid {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	code := r.FormValue("code")
	// The authenticator code is the primary path; a recovery code is the fallback
	// when the authenticator is lost (SignIn delta #314). Both land in the one
	// field, so a failed TOTP falls through to a single-use recovery-code redeem
	// before the login is refused. A 6-digit TOTP never matches a recovery hash and
	// vice versa, so the two never collide.
	if !auth.VerifyTOTP(acct.TotpSecret.String, code, s.now()) && !s.redeemRecoveryCode(r, acct.ID, code) {
		s.render(w, "totp", map[string]any{"Title": "Two-factor", "Error": "Incorrect code."})
		return
	}
	s.clearCookie(w, pendingCookie)
	s.completeLogin(w, r, acct.ID)
}

// redeemRecoveryCode spends a single-use recovery code for the account, reporting
// whether the presented value matched an unused one. It reads the account's still-
// redeemable code hashes and compares the hash of the normalised input; on a match
// it stamps used_at so the code never redeems twice. It never returns which code
// matched, and an empty or non-matching input is simply false — no error path lets
// a comparison failure read as success.
func (s *server) redeemRecoveryCode(r *http.Request, accountID int64, presented string) bool {
	presented = normalizeRecoveryCode(presented)
	if presented == "" {
		return false
	}
	rows, err := s.store.ListUnusedRecoveryCodeHashes(r.Context(), accountID)
	if err != nil {
		log.Printf("web: login: list recovery codes: %v", err)
		return false
	}
	want := hashToken(presented)
	for _, row := range rows {
		if subtleConstantEqual(row.CodeHash, want) {
			if err := s.store.ConsumeRecoveryCode(r.Context(), db.ConsumeRecoveryCodeParams{
				ID: row.ID, UsedAt: s.obsAsOf(),
			}); err != nil {
				log.Printf("web: login: consume recovery code: %v", err)
				return false
			}
			return true
		}
	}
	return false
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	s.clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *server) completeLogin(w http.ResponseWriter, r *http.Request, id int64) {
	if !s.setSignedCookie(w, r, sessionCookie, auth.KindSession, id, s.sessionTTL) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- home / Dashboard -------------------------------------------------------

func (s *server) home(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if r.URL.Path != "/" {
		s.notFound(w, r)
		return
	}
	s.render(w, "home", s.dashboardData(r, acct))
}

// dashVantageView is one provisioned prober shaped for the vantage-health card:
// its name, verified class and current availability.
type dashVantageView struct {
	Name  string
	Class string
	Avail string
}

// dashSignalRow is one rule's current firing census for the open-signal register:
// the rule name, its subject kind and how many subjects it fires on right now. A
// signal carries no severity (CONTEXT.md; signals.go), so this row carries none —
// it is a current-state count, never a scored or timestamped finding.
type dashSignalRow struct {
	Rule  string
	Kind  string
	Fired int
}

// firstRunStep is one step of the empty-estate first-run checklist (#302), shaped
// after FirstRun.jsx's steps array: a number, whether the real read shows it done,
// its title and detail, an optional action (label + href) shown only while the step
// is open, and a Gated flag. Gated step 4 renders a disabled action naming the gate
// rather than a live one — its precondition is a real internet vantage, never a
// fabricated "done".
type firstRunStep struct {
	N           int
	Done        bool
	Title       string
	Detail      string
	ActionLabel string
	ActionHref  string
	Gated       bool
}

// dashboardData assembles the Dashboard's real figures of the shape the example
// composes (KPI band, vantage health, running-scan state, the open-signal
// register). Every read is best-effort: a failure logs and degrades to an em dash
// or an empty region rather than 500ing the landing page a viewer depends on, and
// no figure is fabricated where its datum does not exist.
func (s *server) dashboardData(r *http.Request, acct db.Account) map[string]any {
	ctx := r.Context()

	// Running-scan state — the existing #245 active-dispatch source: a kind is in
	// flight when any recent Dispatch still has ready-or-running jobs.
	active, err := s.activeDispatchKinds(ctx)
	if err != nil {
		log.Printf("web: dashboard: active dispatch kinds: %v", err)
		active = map[string]bool{}
	}

	// Vantage health — the provisioned probers and their availability — plus the
	// currently-unreachable set, which carries the "scans continue" banner.
	var vantages []dashVantageView
	if rows, verr := s.store.ListVantages(ctx); verr == nil {
		for _, v := range rows {
			vantages = append(vantages, dashVantageView{
				Name: v.Name, Class: v.Class, Avail: v.Availability.String,
			})
		}
	} else {
		log.Printf("web: dashboard: list vantages: %v", verr)
	}

	var unavailable []string
	if rows, uerr := s.store.ListUnavailableVantages(ctx); uerr == nil {
		for _, v := range rows {
			unavailable = append(unavailable, v.Name)
		}
	} else {
		log.Printf("web: dashboard: list unavailable vantages: %v", uerr)
	}

	// Estate-size KPIs — current Name and Service subjects (live-tier gated) and the
	// declared scopes. Each carries a Has flag so an unavailable read renders "—",
	// never a fabricated zero.
	names, hasNames := 0, false
	if rows, nerr := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); nerr == nil {
		names, hasNames = len(rows), true
	} else {
		log.Printf("web: dashboard: list name subjects: %v", nerr)
	}

	services, hasServices := 0, false
	if rows, serr := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); serr == nil {
		services, hasServices = len(rows), true
	} else {
		log.Printf("web: dashboard: list service subjects: %v", serr)
	}

	nameScopes, addrScopes, hasScopes := 0, 0, false
	if rows, serr := s.store.ListSeeds(ctx); serr == nil {
		for _, sd := range rows {
			if sd.Kind == "address" {
				addrScopes++
			} else {
				nameScopes++
			}
		}
		hasScopes = true
	} else {
		log.Printf("web: dashboard: list seeds: %v", serr)
	}

	// Open signals — the current count of firing signals and the per-rule firing
	// census. A signal is a current-state census member, so this is the one honest
	// signal figure: no severity to rank, no per-signal recency feed to list. On a
	// corpus failure the signal regions degrade to unavailable rather than 500ing.
	openSignals, hasOpenSignals := 0, false
	var firing []dashSignalRow
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
			if len(c.Fired) > 0 {
				firing = append(firing, dashSignalRow{
					Rule: c.Rule, Kind: signal.SubjectKindFor(c.Rule), Fired: len(c.Fired),
				})
			}
		}
		hasOpenSignals = true
		sort.Slice(firing, func(i, j int) bool {
			if firing[i].Fired != firing[j].Fired {
				return firing[i].Fired > firing[j].Fired
			}
			return firing[i].Rule < firing[j].Rule
		})
	} else {
		log.Printf("web: dashboard: build signal corpus: %v", cerr)
	}

	// First-run state — the home renders the four-step checklist instead of the
	// Dashboard while the estate is empty (#302). "Empty" is the honest read: both
	// the Name and Service censuses resolved and returned nothing, so there is no
	// observed inventory to land on. A failed read does not qualify — the Dashboard
	// then degrades to em dashes rather than mislabelling the estate as first-run.
	emptyEstate := hasNames && hasServices && names == 0 && services == 0

	// Each step is a real read: a scope is declared, a zone file supplied, an
	// internet vantage provisioned, a batch dispatched. Step 4 is gated on the
	// internet vantage (the same signal exposure.go withholds on).
	internetVantage := false
	for _, v := range vantages {
		if v.Class == "internet" {
			internetVantage = true
			break
		}
	}
	zoneUploaded := false
	if rows, zerr := s.store.ListZoneFileStatus(ctx); zerr == nil {
		zoneUploaded = len(rows) > 0
	} else {
		log.Printf("web: dashboard: list zone file status: %v", zerr)
	}
	scanDispatched := false
	if rows, derr := s.store.ListDispatchProgress(ctx, scansHistoryLimit); derr == nil {
		scanDispatched = len(rows) > 0
	} else {
		log.Printf("web: dashboard: list dispatch progress: %v", derr)
	}

	var steps []firstRunStep
	if emptyEstate {
		steps = firstRunChecklist(nameScopes+addrScopes, zoneUploaded, internetVantage, scanDispatched)
	}
	firstRunDone := 0
	for _, st := range steps {
		if st.Done {
			firstRunDone++
		}
	}

	data := map[string]any{
		"Title": "Dashboard", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "dashboard",
		"Scanning":  len(active) > 0,

		"EmptyEstate":   emptyEstate,
		"FirstRunSteps": steps,
		"FirstRunDone":  firstRunDone,

		"Vantages":    vantages,
		"Unavailable": unavailable,

		"OpenSignals":    openSignals,
		"HasOpenSignals": hasOpenSignals,
		"Firing":         firing,

		"Names":       names,
		"HasNames":    hasNames,
		"Services":    services,
		"HasServices": hasServices,
		"NameScopes":  nameScopes,
		"AddrScopes":  addrScopes,
		"Scopes":      nameScopes + addrScopes,
		"HasScopes":   hasScopes,
		"ActiveScans": len(active),
	}
	// Light the nav's Signals pill with the live firing count when there is one.
	if hasOpenSignals && openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	return data
}

// firstRunChecklist builds the four setup steps for the empty-estate home (#302),
// ported from FirstRun.jsx's steps array with the sample data swapped for the real
// reads passed in. Each step's Done is the honest read — never a fabricated done —
// and its action is offered only while the step is open. Step 4 is gated on the
// internet vantage: without one its action is disabled and names the gate, matching
// the withheld/gating pattern exposure.go uses for the same signal.
func firstRunChecklist(scopes int, zoneUploaded, internetVantage, scanDispatched bool) []firstRunStep {
	scopeDetail := "A seed is a boundary, not a starting gun"
	if scopes > 0 {
		unit := "scopes"
		if scopes == 1 {
			unit = "scope"
		}
		scopeDetail = fmt.Sprintf("%d %s declared · a seed is a boundary, not a starting gun", scopes, unit)
	}
	return []firstRunStep{
		{
			N: 1, Done: scopes > 0,
			Title: "Declare your domain", Detail: scopeDetail,
			ActionLabel: "Declare scope", ActionHref: "/scope",
		},
		{
			N: 2, Done: zoneUploaded,
			Title:       "Upload a zone file",
			Detail:      "Enables removal detection — you stopped telling us becomes detectable",
			ActionLabel: "Upload zone", ActionHref: "/scope",
		},
		{
			N: 3, Done: internetVantage,
			Title:       "Add an internet vantage",
			Detail:      "Exposure needs an outside observer, unconditionally",
			ActionLabel: "Provision prober", ActionHref: "/scope",
		},
		{
			N: 4, Done: scanDispatched,
			Title:       "Run the first batch",
			Detail:      "Scans dispatch on cadence; kick the first one now",
			ActionLabel: "Run first batch", ActionHref: "/scans",
			Gated: !internetVantage,
		},
	}
}

// --- account management -----------------------------------------------------

// accountPage redirects the temporary `GET /account` home (#277) to its permanent
// fold in Settings → access (#281). The account details + admin invite/TOTP form
// now live in the Settings access sub-tab, so this route is a redirect and the
// `account` block is gone. The merged SignIn's totp-enroll Cancel link (→/account)
// lands here transparently.
func (s *server) accountPage(w http.ResponseWriter, r *http.Request, _ db.Account) {
	http.Redirect(w, r, "/settings?tab=team", http.StatusSeeOther)
}

func (s *server) createAccount(w http.ResponseWriter, r *http.Request, acct db.Account) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := r.FormValue("role")

	if role != roleAdmin && role != roleViewer {
		s.renderFormError(w, r, acct, "Role must be admin or viewer.")
		return
	}
	if msg := validateCredentials(username, password); msg != "" {
		s.renderFormError(w, r, acct, msg)
		return
	}
	if _, err := s.createAccountRow(r, username, role, password); err != nil {
		s.renderFormError(w, r, acct, createError(err))
		return
	}
	s.renderSettings(w, r, acct, settingsForms{tab: "team", notice: "Account " + username + " created."})
}

func (s *server) totpEnable(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// Do not let an already-enrolled account re-roll its secret through this
	// path: it would set totp_enabled=false and strip the second factor until
	// a fresh confirm — a downgrade a stolen session must not be able to do.
	if acct.TotpEnabled {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		s.serverError(w, "generate totp secret", err)
		return
	}
	if err := s.store.SetTOTPSecret(r.Context(), db.SetTOTPSecretParams{
		ID: acct.ID, TotpSecret: pgtype.Text{String: secret, Valid: true},
	}); err != nil {
		s.serverError(w, "store totp secret", err)
		return
	}
	s.render(w, "totp-enroll", totpEnrollData(acct.Username, secret, ""))
}

// totpEnrollData assembles the template data for the enrollment screen: the
// secret and its otpauth:// URI (always, as the manual-entry fallback) and a
// scannable QR of that URI, rendered in-process. The QR is generated here, not
// by any third-party service, so the secret never leaves the origin (ADR-0053).
// A payload that will not fit a QR (an unusually long username) simply omits the
// image; the secret text carries the enrollment on its own.
func totpEnrollData(username, secret, errMsg string) map[string]any {
	uri := auth.OtpauthURI(secret, username, issuer)
	data := map[string]any{
		"Title": "Two-factor", "Secret": secret, "OtpauthURI": uri,
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if svg, err := qr.SVG([]byte(uri), "Two-factor enrollment QR code for "+username); err == nil {
		data["OtpauthQR"] = template.HTML(svg) //nolint:gosec // SVG is built by our own encoder, not user input
	}
	return data
}

func (s *server) totpConfirm(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fresh, err := s.store.GetAccountByID(r.Context(), acct.ID)
	if err != nil || !fresh.TotpSecret.Valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !auth.VerifyTOTP(fresh.TotpSecret.String, r.FormValue("code"), s.now()) {
		s.render(w, "totp-enroll", totpEnrollData(acct.Username, fresh.TotpSecret.String,
			"Incorrect code. Two-factor is not enabled."))
		return
	}
	if err := s.store.ConfirmTOTP(r.Context(), acct.ID); err != nil {
		s.serverError(w, "confirm totp", err)
		return
	}
	fresh.TotpEnabled = true

	// Two-factor is now on; issue the recovery codes the SignIn delta's enrollment
	// screen reveals once (#314). They are generated, their hashes stored (the
	// plaintext is never persisted), and the plaintext handed back in THIS response
	// only — the finish action is a plain navigation, so a refresh cannot re-show
	// them. Re-issuing clears any prior set so only this set redeems. A failure to
	// store the codes must not leave two-factor on with no recovery path, so it is a
	// hard error rather than a silent skip.
	plain, hashes, err := newRecoveryCodes(recoveryCodeCount)
	if err != nil {
		s.serverError(w, "generate recovery codes", err)
		return
	}
	if err := s.store.DeleteRecoveryCodesForAccount(r.Context(), acct.ID); err != nil {
		s.serverError(w, "clear recovery codes", err)
		return
	}
	for _, h := range hashes {
		if err := s.store.CreateRecoveryCode(r.Context(), db.CreateRecoveryCodeParams{
			AccountID: acct.ID, CodeHash: h,
		}); err != nil {
			s.serverError(w, "store recovery code", err)
			return
		}
	}
	s.render(w, "totp-recovery", map[string]any{"Title": "Two-factor", "Codes": plain})
}

// --- forgot / reset password (#314, T19) ------------------------------------

// forgotForm renders the "enter your account name" step of the reset flow. It is
// pre-auth: a caller who has lost their password has no session to gate on.
func (s *server) forgotForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "forgot", map[string]any{"Title": "Reset password"})
}

// forgotSubmit mints a single-use reset link for the named account, then always
// renders the same "if that account exists, a link is on its way" done state — the
// response is identical whether or not the username exists, so the endpoint reveals
// nothing about which accounts do. There is no mail on a self-hosted host, so the
// link is delivered out of band: it is written to the web logs, exactly as the
// first-boot setup token is, and the operator can also reset directly on the host.
func (s *server) forgotSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	if acct, err := s.store.GetAccountByUsername(r.Context(), username); err == nil {
		if plaintext, hash, terr := newOpaqueToken(); terr != nil {
			log.Printf("web: forgot: mint reset token: %v", terr)
		} else if _, cerr := s.store.CreatePasswordReset(r.Context(), db.CreatePasswordResetParams{
			AccountID: acct.ID, TokenHash: hash,
			ExpiresAt: pgtype.Timestamptz{Time: s.now().Add(s.resetTTL), Valid: true},
		}); cerr != nil {
			log.Printf("web: forgot: create reset: %v", cerr)
		} else {
			// The one delivery this self-hosted build honestly has: the operator's
			// own logs. Never mailed, never shown in the browser response.
			log.Printf("web: password reset requested for %q; set a new password at /reset?token=%s (expires in %s)",
				username, plaintext, s.resetTTL)
		}
	}
	s.render(w, "forgot-sent", map[string]any{"Title": "Reset password"})
}

// resetForm renders the set-a-new-password step for a valid, unspent, unexpired
// reset token, or the honest invalid state when the token is missing, spent, or
// stale — never a form that would fail on submit.
func (s *server) resetForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if _, ok := s.lookupReset(r, token); !ok {
		s.render(w, "reset-invalid", map[string]any{"Title": "Reset password"})
		return
	}
	s.render(w, "reset", map[string]any{"Title": "Set a new password", "Token": token})
}

// resetSubmit sets the account's password from a valid reset token and spends the
// token so the link is single-use. It does not claim to sign other sessions out:
// this build's sessions are stateless signed cookies with no registry, so a session
// elsewhere lapses when it expires rather than being revoked here — the done copy
// says so plainly rather than implying a global sign-out.
func (s *server) resetSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	pr, ok := s.lookupReset(r, token)
	if !ok {
		s.render(w, "reset-invalid", map[string]any{"Title": "Reset password"})
		return
	}
	pw := r.FormValue("password")
	confirm := r.FormValue("confirm")
	fail := func(msg string) {
		s.renderStatus(w, http.StatusBadRequest, "reset", map[string]any{"Title": "Set a new password", "Token": token, "Error": msg})
	}
	if msg := validatePassword(pw); msg != "" {
		fail(msg)
		return
	}
	if pw != confirm {
		fail("Passwords do not match.")
		return
	}
	hash, err := auth.HashPassword(pw)
	if err != nil {
		s.serverError(w, "reset: hash password", err)
		return
	}
	if err := s.store.UpdatePassword(r.Context(), db.UpdatePasswordParams{ID: pr.AccountID, PasswordHash: hash}); err != nil {
		s.serverError(w, "reset: update password", err)
		return
	}
	if err := s.store.ConsumePasswordReset(r.Context(), db.ConsumePasswordResetParams{ID: pr.ID, ConsumedAt: s.obsAsOf()}); err != nil {
		log.Printf("web: reset: consume token: %v", err)
	}
	s.render(w, "reset-done", map[string]any{"Title": "Password updated"})
}

// lookupReset resolves a presented reset token to its row and reports whether it is
// spendable: it must exist, be unconsumed, and be unexpired against the server
// clock. The clock check lives here, not in SQL, so a fixed-clock test and
// production agree on the boundary.
func (s *server) lookupReset(r *http.Request, token string) (db.PasswordReset, bool) {
	if token == "" {
		return db.PasswordReset{}, false
	}
	pr, err := s.store.GetPasswordResetByHash(r.Context(), hashToken(token))
	if err != nil {
		return db.PasswordReset{}, false
	}
	if pr.ConsumedAt.Valid {
		return db.PasswordReset{}, false
	}
	if pr.ExpiresAt.Valid && !pr.ExpiresAt.Time.After(s.now()) {
		return db.PasswordReset{}, false
	}
	return pr, true
}

// --- invite acceptance (#314, T19) ------------------------------------------

// inviteForm renders the set-credentials step for a valid invite token, showing the
// role the new account will hold, or the honest invalid state when the token is
// missing, spent, or expired. Pre-auth by construction: an invitee holds only the
// token, no session. The invite CREATION side (minting a token at a role) lands in
// T18 under Settings -> Team; this is the acceptance half.
func (s *server) inviteForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	inv, ok := s.lookupInvite(r, token)
	if !ok {
		s.render(w, "invite-invalid", map[string]any{"Title": "Invitation"})
		return
	}
	s.render(w, "invite", map[string]any{"Title": "Accept invitation", "Token": token, "Role": inv.Role})
}

// inviteAccept creates the account the invite grants — the acceptor's chosen
// username and password at the invite's role — then spends the invite so the token
// is single-use. It grants no session: the new operator signs in with the
// credentials they just set (landing on /login with a notice), and enrols two-factor
// after first sign-in, so no privileged state is minted straight from a token.
func (s *server) inviteAccept(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	inv, ok := s.lookupInvite(r, token)
	if !ok {
		s.render(w, "invite-invalid", map[string]any{"Title": "Invitation"})
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	fail := func(msg string) {
		s.renderStatus(w, http.StatusBadRequest, "invite", map[string]any{
			"Title": "Accept invitation", "Token": token, "Role": inv.Role,
			"Error": msg, "Username": username,
		})
	}
	if msg := validateCredentials(username, password); msg != "" {
		fail(msg)
		return
	}
	acct, err := s.createAccountRow(r, username, inv.Role, password)
	if err != nil {
		fail(createError(err))
		return
	}
	if err := s.store.ConsumeInvite(r.Context(), db.ConsumeInviteParams{
		ID: inv.ID, ConsumedAt: s.obsAsOf(), AcceptedAccountID: pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		log.Printf("web: invite: consume token: %v", err)
	}
	http.Redirect(w, r, "/login?invited=1", http.StatusSeeOther)
}

// lookupInvite resolves a presented invite token to its row and reports whether it
// is spendable: it must exist, be unconsumed, and be unexpired against the server
// clock — the same discipline lookupReset holds.
func (s *server) lookupInvite(r *http.Request, token string) (db.Invite, bool) {
	if token == "" {
		return db.Invite{}, false
	}
	inv, err := s.store.GetInviteByTokenHash(r.Context(), hashToken(token))
	if err != nil {
		return db.Invite{}, false
	}
	if inv.ConsumedAt.Valid {
		return db.Invite{}, false
	}
	if inv.ExpiresAt.Valid && !inv.ExpiresAt.Time.After(s.now()) {
		return db.Invite{}, false
	}
	if inv.Role != roleAdmin && inv.Role != roleViewer {
		return db.Invite{}, false
	}
	return inv, true
}

// --- pre-auth token helpers (#314, T19) -------------------------------------

// recoveryCodeCount is the number of recovery codes issued at TOTP enrollment,
// matching the SignIn.jsx enrollment screen.
const recoveryCodeCount = 8

// recoveryAlphabet is the character set recovery codes draw from: lowercase letters
// and digits with the visually ambiguous ones (0/o, 1/l/i) removed, so a code read
// off a screen and typed back is hard to mistranscribe.
const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// newOpaqueToken returns a fresh URL-safe token to hand out once and the SHA-256
// hash to store in its place. The plaintext is high-entropy random (256 bits), so a
// digest is the right keeper — unlike a low-entropy password it needs no slow KDF,
// exactly as the personal-token mint reasons.
func newOpaqueToken() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plaintext = base64.RawURLEncoding.EncodeToString(b)
	return plaintext, hashToken(plaintext), nil
}

// hashToken is the one keeper for every pre-auth secret this file mints: the SHA-256
// hex digest of the plaintext. High-entropy tokens and recovery codes are stored
// only as this digest, so a leaked database yields nothing presentable.
func hashToken(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// newRecoveryCodes returns n fresh recovery codes to reveal once and their hashes to
// store. Each code is two groups of four characters (e.g. k4mq-9d2x), the shape the
// SignIn enrollment screen shows. The modulo bias over a 31-character alphabet is
// negligible for a single-use fallback credential.
func newRecoveryCodes(n int) (plain, hashes []string, err error) {
	plain = make([]string, 0, n)
	hashes = make([]string, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 8)
		if _, err = rand.Read(buf); err != nil {
			return nil, nil, err
		}
		var sb strings.Builder
		for j, b := range buf {
			if j == 4 {
				sb.WriteByte('-')
			}
			sb.WriteByte(recoveryAlphabet[int(b)%len(recoveryAlphabet)])
		}
		code := sb.String()
		plain = append(plain, code)
		hashes = append(hashes, hashToken(code))
	}
	return plain, hashes, nil
}

// normalizeRecoveryCode canonicalises a presented recovery code for comparison:
// trimmed and lower-cased, so a code typed with stray whitespace or in upper case
// still redeems. The dash is kept, since the stored hash is of the dashed form.
func normalizeRecoveryCode(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// subtleConstantEqual compares two hex digests in constant time, so a recovery-code
// redeem does not leak how far a near-miss matched through timing.
func subtleConstantEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// --- profile (#304, T9) -----------------------------------------------------

// profileTokenView is one personal API token shaped for the tokens table: its id
// (for the revoke link), the operator's label, the non-secret prefix, and the two
// timestamps. Last is an em dash for a token never yet presented — the honest read
// of last_used_at IS NULL, never a fabricated recency.
type profileTokenView struct {
	ID      int64
	Name    string
	Prefix  string
	Created string
	Last    string
}

// profileState carries the transient, per-request Profile surface: which dialog is
// open, a freshly minted token to reveal once, and any inline errors. It never
// holds persisted data — that is read fresh in renderProfile — so a plain page load
// passes the zero value.
type profileState struct {
	notice     string
	pwError    string
	createOpen bool
	tokError   string
	tokName    string
	minted     string // freshly minted plaintext — shown once, never stored
	mintedName string
	revokeID   int64 // token-revoke ConfirmDialog target; 0 = closed
	revokeErr  string
	endSession bool   // end-session ConfirmDialog open
	ssoNotice  string // SSO link/unlink outcome (a success or benign message)
	ssoError   string // SSO link failure (a refusal)
}

// profilePage renders the account's own Profile (#304): identity, credentials with
// the 2FA status, the current session, and the personal API tokens. It is
// viewer-readable — a Profile is personal, so any signed-in account manages its own
// credentials and tokens. The create/revoke/end-session dialogs are opened by query
// param so the destructive ones are a navigation, never a click that fires the act.
func (s *server) profilePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	st := profileState{}
	q := r.URL.Query()
	if q.Get("new") != "" {
		st.createOpen = true
	}
	if v := q.Get("revoke"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			st.revokeID = id
		}
	}
	if q.Get("endsession") != "" {
		st.endSession = true
	}
	if q.Get("saved") != "" {
		st.notice = "Password changed. Other sessions keep working until they expire."
	}
	// SSO self-link outcomes ride back as fixed query codes (never reflected free text),
	// each mapped here to an honest message.
	switch q.Get("linked") {
	case "1":
		st.ssoNotice = "Identity linked. You can now sign in with it."
	case "exists":
		st.ssoNotice = "That identity is already linked to your account."
	}
	if q.Get("unlinked") != "" {
		st.ssoNotice = "Identity unlinked. It can no longer sign in to this account."
	}
	switch q.Get("linkerr") {
	case "provider":
		st.ssoError = "You already have an identity linked for that provider. Unlink it first to link a different one."
	case "elsewhere":
		st.ssoError = "That identity is already linked to another account."
	case "cancelled":
		st.ssoError = "Linking was cancelled or refused."
	case "expired":
		st.ssoError = "That link attempt expired. Try again."
	case "unavailable":
		st.ssoError = "That identity provider could not be reached. Try again."
	case "failed":
		st.ssoError = "Linking could not be completed. Try again."
	}
	s.renderProfile(w, r, acct, st)
}

// renderProfile assembles the Profile page's real data of the shape Profile.jsx
// composes and renders it with the transient state. Every figure is a real read of
// this account: username and role from the row, the 2FA status from totp_enabled,
// the current session from the live request, and the tokens from the store. No
// sample datum survives.
func (s *server) renderProfile(w http.ResponseWriter, r *http.Request, acct db.Account, st profileState) {
	// Read the account fresh so the 2FA status reflects an enrolment completed in
	// this same session (the shared totp flow mutates the row, not this acct copy).
	if fresh, err := s.store.GetAccountByID(r.Context(), acct.ID); err == nil {
		acct = fresh
	}

	var tokens []profileTokenView
	if rows, err := s.store.ListPersonalTokens(r.Context(), acct.ID); err == nil {
		for _, t := range rows {
			tokens = append(tokens, profileTokenView{
				ID: t.ID, Name: t.Name, Prefix: t.Prefix,
				Created: isoDate(t.CreatedAt), Last: lastUsed(t.LastUsedAt),
			})
		}
	} else {
		log.Printf("web: profile: list personal tokens: %v", err)
	}

	// SSO (#319, ADR-0113): the account's own linked identities, and the enabled
	// providers not yet linked (each offers a "Link" button). A read failure degrades to
	// an empty surface rather than failing the whole Profile; when the linked list could
	// not be read, the link offer is suppressed too, so a blip never invites re-linking an
	// already-linked provider.
	linked, linkedProviders, ok := s.profileSSOIdentities(r, acct.ID)
	var available []profileLinkView
	if ok {
		available = s.profileLinkableProviders(r, linkedProviders)
	}

	// The revoke ConfirmDialog names its target; resolve it from the read so a stale
	// or foreign id simply renders no dialog rather than a gate with no subject.
	revokeName := ""
	if st.revokeID != 0 {
		for _, t := range tokens {
			if t.ID == st.revokeID {
				revokeName = t.Name
				break
			}
		}
		if revokeName == "" {
			st.revokeID = 0
		}
	}

	data := map[string]any{
		"Title": "Profile", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "",

		"Initials":    initials(acct.Username),
		"Username":    acct.Username,
		"Role":        acct.Role,
		"CreatedISO":  isoDate(acct.CreatedAt),
		"TotpEnabled": acct.TotpEnabled,

		"SessionDevice": sessionDevice(r),
		"SessionIP":     sessionIP(r),

		"Tokens": tokens,

		"SSOIdentities": linked,
		"SSOProviders":  available,
		"SSONotice":     st.ssoNotice,
		"SSOError":      st.ssoError,

		"Notice":     st.notice,
		"PwError":    st.pwError,
		"CreateOpen": st.createOpen,
		"TokError":   st.tokError,
		"TokName":    st.tokName,
		"Minted":     st.minted,
		"MintedName": st.mintedName,
		"RevokeID":   st.revokeID,
		"RevokeName": revokeName,
		"RevokeErr":  st.revokeErr,
		"EndSession": st.endSession,
	}
	s.render(w, "profile", data)
}

// profileIdentityView is one linked SSO identity shown on the Profile: the binding id
// (for the unlink form), the provider it came through, and the display label captured at
// link time. The subject is never surfaced — it is opaque and of no use to a human.
type profileIdentityView struct {
	ID          int64
	Provider    string
	DisplayName string
	LinkedAt    string
}

// profileLinkView is an enabled provider the account has not yet linked — the Profile
// renders a "Link" button per entry.
type profileLinkView struct {
	Slug string
	Name string
}

// profileSSOIdentities reads the account's linked identities for its Profile, returning
// the display views, the set of provider ids already linked (so the linkable list can
// exclude them), and ok=false on a read failure. A failure degrades to an empty surface
// rather than failing the page — and the caller then suppresses the link offer too, so a
// blip never invites re-linking a provider the account has already linked.
func (s *server) profileSSOIdentities(r *http.Request, accountID int64) ([]profileIdentityView, map[int64]bool, bool) {
	rows, err := s.store.ListSSOIdentitiesForAccount(r.Context(), accountID)
	if err != nil {
		log.Printf("web: profile: list sso identities: %v", err)
		return nil, map[int64]bool{}, false
	}
	linkedProviders := make(map[int64]bool, len(rows))
	out := make([]profileIdentityView, 0, len(rows))
	for _, row := range rows {
		linkedProviders[row.ProviderID] = true
		out = append(out, profileIdentityView{
			ID: row.ID, Provider: row.ProviderName, DisplayName: row.DisplayName,
			LinkedAt: isoDate(row.CreatedAt),
		})
	}
	return out, linkedProviders, true
}

// profileLinkableProviders lists the enabled providers the account has not yet linked.
// An account holds at most one identity per provider, so a provider already linked drops
// out of the offer.
func (s *server) profileLinkableProviders(r *http.Request, linked map[int64]bool) []profileLinkView {
	rows, err := s.store.ListEnabledSSOProviders(r.Context())
	if err != nil {
		log.Printf("web: profile: list enabled sso providers: %v", err)
		return nil
	}
	out := make([]profileLinkView, 0, len(rows))
	for _, p := range rows {
		if linked[p.ID] {
			continue
		}
		out = append(out, profileLinkView{Slug: p.Slug, Name: p.Name})
	}
	return out
}

// changePassword is the self-service password change (Profile → Credentials). It
// verifies the current password against a fresh read, enforces the same length
// bounds as every other credential path, and never touches the TOTP secret — a
// password change leaves the second factor in force. Other sessions are not
// invalidated (this build's sessions are stateless signed cookies), which the
// success notice states plainly rather than implying a global sign-out.
func (s *server) changePassword(w http.ResponseWriter, r *http.Request, acct db.Account) {
	current := r.FormValue("current_password")
	next := r.FormValue("new_password")

	fresh, err := s.store.GetAccountByID(r.Context(), acct.ID)
	if err != nil {
		s.serverError(w, "profile: read account", err)
		return
	}
	if !auth.CheckPassword(fresh.PasswordHash, current) {
		s.renderProfile(w, r, acct, profileState{pwError: "Current password is incorrect."})
		return
	}
	if msg := validatePassword(next); msg != "" {
		s.renderProfile(w, r, acct, profileState{pwError: msg})
		return
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		s.serverError(w, "profile: hash password", err)
		return
	}
	if err := s.store.UpdatePassword(r.Context(), db.UpdatePasswordParams{ID: acct.ID, PasswordHash: hash}); err != nil {
		s.serverError(w, "profile: update password", err)
		return
	}
	http.Redirect(w, r, "/profile?saved=1", http.StatusSeeOther)
}

// createPersonalToken mints a personal API token and reveals it once. The plaintext
// is generated, its hash stored, and the plaintext handed back in THIS response
// only — the page is rendered directly (not redirected) so the value can be shown a
// single time; a refresh re-GETs /profile without it. Verge keeps only the hash.
func (s *server) createPersonalToken(w http.ResponseWriter, r *http.Request, acct db.Account) {
	name := strings.TrimSpace(r.FormValue("name"))
	switch {
	case name == "":
		s.renderProfile(w, r, acct, profileState{createOpen: true, tokError: "Give the token a name.", tokName: name})
		return
	case len(name) > 64:
		s.renderProfile(w, r, acct, profileState{createOpen: true, tokError: "Name must be 64 characters or fewer.", tokName: name})
		return
	}
	plaintext, prefix, hash, err := mintPersonalToken()
	if err != nil {
		s.serverError(w, "profile: mint token", err)
		return
	}
	if _, err := s.store.CreatePersonalToken(r.Context(), db.CreatePersonalTokenParams{
		AccountID: acct.ID, Name: name, Prefix: prefix, TokenHash: hash,
	}); err != nil {
		if isUniqueViolation(err) {
			s.renderProfile(w, r, acct, profileState{createOpen: true, tokError: "You already have a token named that.", tokName: name})
			return
		}
		s.serverError(w, "profile: create token", err)
		return
	}
	s.renderProfile(w, r, acct, profileState{minted: plaintext, mintedName: name})
}

// revokePersonalToken revokes a token through the typed-name gate: the operator must
// type the token's exact name to confirm, guarding the worst destructive act on the
// page — a revoke is irreversible and silently breaks whatever automation held the
// token. It is reached only through the ConfirmDialog (a POST), never a menu click,
// and the delete is scoped to the owner so no account can revoke another's token.
func (s *server) revokePersonalToken(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	typed := strings.TrimSpace(r.FormValue("confirm_name"))

	rows, err := s.store.ListPersonalTokens(r.Context(), acct.ID)
	if err != nil {
		s.serverError(w, "profile: list tokens", err)
		return
	}
	name := ""
	for _, t := range rows {
		if t.ID == id {
			name = t.Name
			break
		}
	}
	if name == "" {
		// Already gone (or never this account's) — nothing to revoke.
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if typed != name {
		s.renderProfile(w, r, acct, profileState{
			revokeID:  id,
			revokeErr: "That did not match. Type " + name + " exactly to revoke.",
		})
		return
	}
	if err := s.store.DeletePersonalToken(r.Context(), db.DeletePersonalTokenParams{ID: id, AccountID: acct.ID}); err != nil {
		s.serverError(w, "profile: revoke token", err)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// revokeSession ends the current session. This build's sessions are stateless signed
// cookies with no server-side registry, so the one session honestly revocable is the
// one making the request: revoking it clears the cookie and lands on /login, exactly
// as sign-out does. It is reached only through the end-session ConfirmDialog.
func (s *server) revokeSession(w http.ResponseWriter, r *http.Request, _ db.Account) {
	s.clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// mintPersonalToken generates a personal token, returning the plaintext to reveal
// once, the non-secret prefix to store for display, and the hash to store in place
// of the secret. The token is high-entropy random, so a SHA-256 digest is the right
// keeper — unlike a low-entropy password, it needs no slow KDF.
func mintPersonalToken() (plaintext, prefix, hash string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	plaintext = "vg_pat_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	prefix = plaintext[:11] + "…" // vg_pat_ + 4 hex chars + ellipsis
	return plaintext, prefix, hash, nil
}

// validatePassword bounds a new password to the same range every credential path
// enforces: at least 8 characters, and at most 72 (bcrypt hashes no more).
func validatePassword(pw string) string {
	switch {
	case len(pw) < 8:
		return "Password must be at least 8 characters."
	case len(pw) > 72:
		return "Password must be 72 characters or fewer."
	default:
		return ""
	}
}

// isoDate formats a timestamp as an ISO 8601 date, or an em dash when absent.
func isoDate(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return "—"
	}
	return ts.Time.Format("2006-01-02")
}

// lastUsed renders a token's last-used instant, or an em dash when it has never been
// presented — the honest read of a NULL last_used_at, not a fabricated recency.
func lastUsed(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return "—"
	}
	return ts.Time.Format("2006-01-02")
}

// initials derives a two-letter avatar label from the username, upper-cased.
func initials(username string) string {
	u := strings.TrimSpace(username)
	if u == "" {
		return "?"
	}
	r := []rune(strings.ToUpper(u))
	if len(r) == 1 {
		return string(r[0])
	}
	return string(r[0]) + string(r[1])
}

// sessionDevice describes the current session from the request's User-Agent — a real
// derivation of what the client sent, never a fabricated device. An unrecognised or
// absent agent degrades to a plain label rather than a guess.
func sessionDevice(r *http.Request) string {
	ua := r.UserAgent()
	if ua == "" {
		return "This session"
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

// sessionIP is the address this request arrived from. It reads RemoteAddr, never a
// proxy-supplied forwarding header — the same rule the auth path holds: a header is
// never trusted, here not even for display.
func sessionIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- helpers ---------------------------------------------------------------

func (s *server) createAccountRow(r *http.Request, username, role, password string) (db.Account, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	return s.store.CreateAccount(r.Context(), db.CreateAccountParams{
		Username: username, Role: role, PasswordHash: hash,
	})
}

// validateCredentials returns a user-facing message, or "" when acceptable.
func validateCredentials(username, password string) string {
	switch {
	case username == "":
		return "Username is required."
	case len(username) > 64:
		return "Username must be 64 characters or fewer."
	case len(password) < 8:
		return "Password must be at least 8 characters."
	case len(password) > 72:
		// bcrypt hashes at most 72 bytes and errors beyond that; reject it
		// here with a clear message rather than a generic create failure.
		return "Password must be 72 characters or fewer."
	default:
		return ""
	}
}

// createError maps a CreateAccount failure to a user-facing message. A unique
// violation means the username is taken; everything else is opaque.
func createError(err error) string {
	if isUniqueViolation(err) {
		return "That username is already taken."
	}
	return "Could not create the account."
}

func isUniqueViolation(err error) bool {
	var e interface{ SQLState() string }
	return errors.As(err, &e) && e.SQLState() == "23505"
}

// isForeignKeyViolation reports a 23503 — a referencing row still points at this
// one. Removing a member hits it when the account authored attributed acts (a
// NOT NULL created_by), so the handler turns it into a clear refusal rather than a
// 500.
func isForeignKeyViolation(err error) bool {
	var e interface{ SQLState() string }
	return errors.As(err, &e) && e.SQLState() == "23503"
}

// renderFormError re-renders the Settings Team sub-tab with the form's error and a
// 400, so a rejected /accounts POST echoes its message where account management now
// lives (#281, retargeted to Team in #313).
func (s *server) renderFormError(w http.ResponseWriter, r *http.Request, acct db.Account, msg string) {
	s.renderSettings(w, r, acct, settingsForms{section: "team", teamError: msg})
}

func (s *server) render(w http.ResponseWriter, name string, data any) {
	s.renderStatus(w, http.StatusOK, name, data)
}

// renderStatus writes an HTML page at the given status. The Content-Type must
// be set before WriteHeader commits the header block, or it is silently
// dropped and the browser renders the markup as plain text.
func (s *server) renderStatus(w http.ResponseWriter, status int, name string, data any) {
	s.injectUnread(data)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("web: render %s: %v", name, err)
	}
}

// injectUnread supplies the chrome's global message element with the unread
// count on every authenticated screen, so no per-page handler has to thread it
// through (v1 spec §6.1: the count rides every screen). It is a single central
// touchpoint: a chrome page is any render whose data carries an "IsAdmin" key —
// the auth pages (login/setup/totp) have no chrome and no such key, so they are
// left alone. A page that already computed its own "Unread" (the panel itself)
// keeps it. The count is a lightweight read; on error it defaults to zero rather
// than failing the page.
func (s *server) injectUnread(data any) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}
	if _, isChrome := m["IsAdmin"]; !isChrome {
		return
	}
	// The shared chrome highlights the active nav pill via {{if eq .NavActive "id"}}.
	// A missing map key would make `eq` error at render, so every chrome page gets
	// a default here; a screen handler that passes its own "NavActive" nav id keeps
	// it. (T0 seam: the nav id is the one field a screen ticket threads into the
	// shell. The key is "NavActive", not "Active" — the scans view already owns
	// "Active" for its in-flight list.)
	if _, has := m["NavActive"]; !has {
		m["NavActive"] = ""
	}
	if _, has := m["Unread"]; has {
		return
	}
	n, err := s.store.CountUnreadMessages(context.Background())
	if err != nil {
		log.Printf("web: unread count: %v", err)
	}
	m["Unread"] = n
}

func (s *server) serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("web: %s: %v", what, err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// setSignedCookie writes a signed cookie of the given kind for the account. It
// is HttpOnly and SameSite=Lax (which blocks cross-site POSTs, the CSRF vector
// for the mutating endpoints), and Secure when the request arrived over TLS.
// It reports success: on a signing failure it has already written a 500, and
// the caller must return rather than write a second response.
func (s *server) setSignedCookie(w http.ResponseWriter, r *http.Request, name string, kind auth.Kind, id int64, ttl time.Duration) bool {
	token, err := auth.SignSession(s.key, auth.Session{
		AccountID: id, Kind: kind, ExpiresAt: s.now().Add(ttl),
	})
	if err != nil {
		s.serverError(w, "sign session", err)
		return false
	}
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.secureCookies || r.TLS != nil, MaxAge: int(ttl.Seconds()),
	})
	return true
}

func (s *server) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}
