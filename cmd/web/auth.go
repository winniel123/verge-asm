package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
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
	s.render(w, "login", map[string]any{"Title": "Sign in"})
}

func (s *server) loginSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		auth.CheckPassword(dummyHash, password) // equalise timing with the found path
		s.render(w, "login", map[string]any{"Title": "Sign in", "Error": "Invalid username or password."})
		return
	}
	if !auth.CheckPassword(acct.PasswordHash, password) {
		s.render(w, "login", map[string]any{"Title": "Sign in", "Error": "Invalid username or password."})
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
	if !auth.VerifyTOTP(acct.TotpSecret.String, r.FormValue("code"), s.now()) {
		s.render(w, "totp", map[string]any{"Title": "Two-factor", "Error": "Incorrect code."})
		return
	}
	s.clearCookie(w, pendingCookie)
	s.completeLogin(w, r, acct.ID)
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
	http.Redirect(w, r, "/settings?tab=access", http.StatusSeeOther)
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
	s.renderSettings(w, r, acct, settingsForms{tab: "access", notice: "Account " + username + " created."})
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
	s.render(w, "totp-enroll", map[string]any{
		"Title": "Two-factor", "Secret": secret,
		"OtpauthURI": auth.OtpauthURI(secret, acct.Username, issuer),
	})
}

func (s *server) totpConfirm(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fresh, err := s.store.GetAccountByID(r.Context(), acct.ID)
	if err != nil || !fresh.TotpSecret.Valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !auth.VerifyTOTP(fresh.TotpSecret.String, r.FormValue("code"), s.now()) {
		s.render(w, "totp-enroll", map[string]any{
			"Title": "Two-factor", "Secret": fresh.TotpSecret.String,
			"OtpauthURI": auth.OtpauthURI(fresh.TotpSecret.String, acct.Username, issuer),
			"Error":      "Incorrect code. Two-factor is not enabled.",
		})
		return
	}
	if err := s.store.ConfirmTOTP(r.Context(), acct.ID); err != nil {
		s.serverError(w, "confirm totp", err)
		return
	}
	fresh.TotpEnabled = true
	s.renderSettings(w, r, fresh, settingsForms{tab: "access", notice: "Two-factor is now enabled."})
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

// renderFormError re-renders the Settings access sub-tab with the invite form's
// error and a 400, so a rejected /accounts POST echoes its message where the form
// now lives (#281).
func (s *server) renderFormError(w http.ResponseWriter, r *http.Request, acct db.Account, msg string) {
	s.renderSettings(w, r, acct, settingsForms{section: "accounts", acctError: msg})
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
