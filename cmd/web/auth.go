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
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/env"
	"github.com/winniel123/verge-asm/internal/qr"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/signal"
)

const (
	sessionCookie = "verge_session"
	pendingCookie = "verge_totp_pending"
	issuer        = "Verge ASM"
)

// A missing account must cost the same bcrypt work, so timing never enumerates usernames.

var dummyHash, _ = auth.HashPassword("verge-timing-equaliser")

type authedHandler func(w http.ResponseWriter, r *http.Request, acct db.Account)

func (s *server) currentAccount(r *http.Request) (db.Account, bool) {
	// Header-trust forward-auth is a refused bypass class (docs/spec/v1-spec.md §4.3).
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return db.Account{}, false
	}
	sess, err := auth.VerifySession(s.key, c.Value, auth.KindSession, s.now())
	if err != nil {
		return db.Account{}, false
	}
	// A pre-registry cookie carries an empty token, so it resolves no row and is refused (ADR-0117).
	ctx := r.Context()
	row, err := s.store.GetSessionByTokenHash(ctx, db.GetSessionByTokenHashParams{
		TokenHash: hashToken(sess.Token),
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	})
	if err != nil {
		return db.Account{}, false
	}
	acct, err := s.store.GetAccountByID(ctx, row.AccountID)
	if err != nil {
		return db.Account{}, false
	}
	// Touching on every request would amplify one write per request onto a busy session.
	if s.now().Sub(row.LastSeenAt.Time) > time.Minute {
		if err := s.store.TouchSession(ctx, db.TouchSessionParams{
			ID:         row.ID,
			LastSeenAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
		}); err != nil {
			log.Printf("web: touch session: %v", err)
		}
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

func (s *server) requireSettingsAdmin(h authedHandler) http.HandlerFunc {
	return s.requireLogin(func(w http.ResponseWriter, r *http.Request, acct db.Account) {
		// A viewer may read the API-access tab; every other Settings tab stays admin-only.
		if acct.Role != roleAdmin && validTab(r.URL.Query().Get("tab")) != "api" {
			s.settingsForbidden(w, r, acct)
			return
		}
		h(w, r, acct)
	})
}

const (
	roleAdmin  = "admin"
	roleViewer = "viewer"
)

func (s *server) setupForm(w http.ResponseWriter, r *http.Request) {
	if s.setupClosed(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.render(w, r, "setup", s.signinData(map[string]any{"Title": "Setup", "Token": r.URL.Query().Get("token")}))
}

func (s *server) setupSubmit(w http.ResponseWriter, r *http.Request) {
	// Without this, two valid first-boot POSTs each observe zero accounts and each create an admin.
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
		s.render(w, r, "setup", s.signinData(map[string]any{"Title": "Setup", "Token": token, "Error": "Invalid setup token."}))
		return
	}
	if msg := validateCredentials(username, password); msg != "" {
		s.render(w, r, "setup", s.signinData(map[string]any{"Title": "Setup", "Token": token, "Error": msg}))
		return
	}
	if _, err := s.createAccountRow(r, username, roleAdmin, password); err != nil {
		s.render(w, r, "setup", s.signinData(map[string]any{"Title": "Setup", "Token": token, "Error": createError(err)}))
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

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

// These templates cross-reference each other's defines, so every one must parse into this set.

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/signin.tmpl"))

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/setup.tmpl"))

func (s *server) buildVersion() string {
	if s.devMode {
		return devFixtureVersion
	}
	return env.OrDefault("VERGE_VERSION", "dev")
}

func (s *server) signinData(data map[string]any) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	data["Version"] = s.buildVersion()
	return data
}

type ssoLoginProvider struct {
	Slug string
	Name string
	Mark string
}

func ssoMark(name string) string {
	mark := ""
	for _, field := range strings.Fields(name) {
		r := []rune(field)
		if len(r) == 0 {
			continue
		}
		mark += strings.ToUpper(string(r[0]))
		if len(mark) == 2 {
			break
		}
	}
	return mark
}

func (s *server) loginProviders(ctx context.Context, noSSO bool) []ssoLoginProvider {
	if noSSO {
		return nil
	}
	if s.devMode {
		out := make([]ssoLoginProvider, 0, len(devSigninProviders))
		for _, p := range devSigninProviders {
			out = append(out, ssoLoginProvider{Slug: p.slug, Name: p.name, Mark: p.mark})
		}
		return out
	}
	rows := s.enabledSSOProviders(ctx)
	out := make([]ssoLoginProvider, 0, len(rows))
	for _, row := range rows {
		out = append(out, ssoLoginProvider{Slug: row.Slug, Name: row.Name, Mark: ssoMark(row.Name)})
	}
	return out
}

func (s *server) loginForm(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentAccount(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	noSSO := s.devMode && r.URL.Query().Get("variant") == "no-sso"
	data := map[string]any{"Title": "Sign in", "SSOProviders": s.loginProviders(r.Context(), noSSO)}
	if r.URL.Query().Get("invited") != "" {
		data["Notice"] = "Account created. Sign in with your new credentials."
	}
	s.render(w, r, "login", s.signinData(data))
}

func (s *server) loginSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	// Refusing before any password work bounds an online guess per account and per source (#322).
	acctKey, ipKey := loginAccountKey(username), s.loginIPKey(r)
	if s.loginLimiter.locked(acctKey, ipKey) {
		s.render(w, r, "login", s.loginData(r.Context(), lockoutMessage))
		return
	}

	acct, err := s.store.GetAccountByUsername(r.Context(), username)
	if err != nil {
		auth.CheckPassword(dummyHash, password)
		s.loginLimiter.fail(acctKey, ipKey)
		s.render(w, r, "login", s.loginData(r.Context(), "Invalid username or password."))
		return
	}
	if !auth.CheckPassword(acct.PasswordHash, password) {
		s.loginLimiter.fail(acctKey, ipKey)
		s.render(w, r, "login", s.loginData(r.Context(), "Invalid username or password."))
		return
	}
	s.loginLimiter.reset(acctKey, ipKey)
	if acct.TotpEnabled {
		if !s.setSignedCookie(w, r, pendingCookie, auth.KindPending, acct.ID, "", s.pendingTTL) {
			return
		}
		s.render(w, r, "totp", s.signinData(map[string]any{"Title": "Two-factor", "Username": acct.Username}))
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

	// A 6-digit code is brute-forceable, so the second factor is throttled like the first (#322).
	acctKey, ipKey := loginAccountKey(acct.Username), s.loginIPKey(r)
	if s.loginLimiter.locked(acctKey, ipKey) {
		s.clearCookie(w, pendingCookie)
		s.render(w, r, "totp", s.signinData(map[string]any{"Title": "Two-factor", "Username": acct.Username, "Error": lockoutMessage}))
		return
	}

	code := r.FormValue("code")
	// A dev-only bypass of the second factor, gated to VERGE_DEV and unreachable in a real build.
	if s.devMode && code == devFixtureTOTPCode {
		s.loginLimiter.reset(acctKey, ipKey)
		s.clearCookie(w, pendingCookie)
		s.completeLogin(w, r, acct.ID)
		return
	}
	secret, derr := auth.DecryptTOTPSecret(s.totpKey, acct.TotpSecret.String)
	if derr != nil {
		s.serverError(w, "decrypt totp secret", derr)
		return
	}
	step, totpOK := auth.VerifyTOTPStep(secret, code, s.now())
	// Two concurrent logins on one code must not both win (RFC 6238 §5.2, #339).
	if totpOK {
		n, serr := s.store.SetTOTPLastStep(r.Context(), db.SetTOTPLastStepParams{
			ID: acct.ID, TotpLastStep: pgtype.Int8{Int64: step, Valid: true},
		})
		if serr != nil {
			s.serverError(w, "advance totp step", serr)
			return
		}
		if n != 1 {
			totpOK = false
		}
	}
	if !totpOK && !s.redeemRecoveryCode(r, acct.ID, code) {
		// Clearing the pending grant on lockout forces an attacker back to the password step (#322).
		if nowLocked := s.loginLimiter.fail(acctKey, ipKey); nowLocked {
			s.clearCookie(w, pendingCookie)
			s.render(w, r, "totp", s.signinData(map[string]any{"Title": "Two-factor", "Username": acct.Username, "Error": lockoutMessage}))
			return
		}
		s.render(w, r, "totp", s.signinData(map[string]any{"Title": "Two-factor", "Username": acct.Username, "Error": "Incorrect code."}))
		return
	}
	s.loginLimiter.reset(acctKey, ipKey)
	s.clearCookie(w, pendingCookie)
	s.completeLogin(w, r, acct.ID)
}

// Vague by design: naming the tripped key or the wait would let an attacker tune (#322).

const lockoutMessage = "Too many attempts. Try again in a few minutes."

func loginAccountKey(username string) string { return "acct:" + strings.ToLower(username) }

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
	for _, row := range rows {
		if auth.CheckPassword(row.CodeHash, presented) {
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
	// Clearing the cookie alone leaves a copied cookie usable, so the row is revoked too (ADR-0117).
	s.revokeCurrentSession(r)
	s.clearCookie(w, sessionCookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *server) revokeCurrentSession(r *http.Request) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return
	}
	sess, err := auth.VerifySession(s.key, c.Value, auth.KindSession, s.now())
	if err != nil {
		return
	}
	ctx := r.Context()
	row, err := s.store.GetSessionByTokenHash(ctx, db.GetSessionByTokenHashParams{
		TokenHash: hashToken(sess.Token),
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	})
	if err != nil {
		return
	}
	if err := s.store.RevokeSession(ctx, db.RevokeSessionParams{
		ID:        row.ID,
		AccountID: row.AccountID,
		RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		log.Printf("web: revoke session: %v", err)
	}
}

func (s *server) currentSessionID(r *http.Request) (int64, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return 0, false
	}
	sess, err := auth.VerifySession(s.key, c.Value, auth.KindSession, s.now())
	if err != nil {
		return 0, false
	}
	row, err := s.store.GetSessionByTokenHash(r.Context(), db.GetSessionByTokenHashParams{
		TokenHash: hashToken(sess.Token),
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	})
	if err != nil {
		return 0, false
	}
	return row.ID, true
}

func (s *server) completeLogin(w http.ResponseWriter, r *http.Request, id int64) {
	plaintext, hash, err := newOpaqueToken()
	if err != nil {
		s.serverError(w, "mint session token", err)
		return
	}
	if _, err := s.store.CreateSession(r.Context(), db.CreateSessionParams{
		AccountID: id,
		TokenHash: hash,
		UserAgent: r.UserAgent(),
		Ip:        sessionIP(r),
		ExpiresAt: pgtype.Timestamptz{Time: s.now().Add(s.sessionTTL), Valid: true},
	}); err != nil {
		// A cookie whose session has no row can never validate, so this fails closed instead.
		s.serverError(w, "create session", err)
		return
	}
	if !s.setSignedCookie(w, r, sessionCookie, auth.KindSession, id, plaintext, s.sessionTTL) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/dashboard.tmpl"))

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/firstrun.tmpl"))

func (s *server) home(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if r.URL.Path != "/" {
		s.notFound(w, r)
		return
	}
	if s.devMode {
		if r.URL.Query().Get("variant") == devFirstRunVariant {
			s.render(w, r, "home", s.firstRunFixtureData(acct))
			return
		}
		s.render(w, r, "home", s.dashboardFixtureData(acct, r))
		return
	}
	s.render(w, r, "home", s.dashboardData(r, acct))
}

type dashVantageView struct {
	Name    string
	Class   string
	Avail   string
	Latency string
}

type dashSevBar struct {
	Sev   string
	Count int
	Pct   int
}

type dashRecentSignal struct {
	Severity string
	SevLabel string
	Title    string
	Asset    string
	Port     string
	Seen     string
	ViewKey  string
}

type dashSilentZone struct {
	Bound string
	Text  string
}

type dashStat struct {
	Label    string
	Value    string
	Caption  string
	Live     bool
	HasDelta bool
	Change   int
	Tone     string
}

type firstRunStep struct {
	Num         int
	Done        bool
	Title       string
	Detail      string
	HasAction   bool
	ActionLabel string
	ActionHref  string
	ActionPost  string
	Gated       bool
	GateTitle   string
}

func (s *server) dashboardData(r *http.Request, acct db.Account) map[string]any {
	ctx := r.Context()

	active, err := s.activeDispatchKinds(ctx)
	if err != nil {
		log.Printf("web: dashboard: active dispatch kinds: %v", err)
		active = map[string]bool{}
	}

	// Failing closed here over-reports a vantage as internet rather than mislabelling it internal.
	covered, cerr := s.addressScopeCovered(ctx)
	if cerr != nil {
		log.Printf("web: dashboard: address scope coverage: %v", cerr)
		covered = func(netip.Addr) bool { return false }
	}
	var vantages []dashVantageView
	// The vestigial class column is never read; the chip is derived per read (#709).
	if rows, verr := s.store.ListVantages(ctx); verr == nil {
		for _, v := range rows {
			vantages = append(vantages, dashVantageView{
				Name: v.Name, Class: string(vantageFactsClass(v.DialledAddr, v.Egress, covered)),
				Avail:   v.Availability.String,
				Latency: vantageLatencyLabel(v.LatencyMs),
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

	names, hasNames := 0, false
	if rows, nerr := s.store.ListCurrentNameSubjects(ctx, db.ListCurrentNameSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); nerr == nil {
		names, hasNames = len(rows), true
	} else {
		log.Printf("web: dashboard: list name subjects: %v", nerr)
	}

	services, hasServices := 0, false
	var walked []walkedAddr
	if rows, serr := s.store.ListCurrentServiceSubjects(ctx, db.ListCurrentServiceSubjectsParams{
		Search: "", AsOf: s.obsAsOf(), FloorCadences: retention.FloorCadences,
	}); serr == nil {
		services, hasServices = len(rows), true
		walked = walkedAddresses(rows)
	} else {
		log.Printf("web: dashboard: list service subjects: %v", serr)
	}

	nameScopes, addrScopes, hasScopes := 0, 0, false
	var seedRows []db.ListSeedsRow
	if rows, serr := s.store.ListSeeds(ctx); serr == nil {
		seedRows = rows
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

	openSignals, hasOpenSignals := 0, false
	criticalSignals := 0
	sevCounts := map[string]int{}
	var recentSignals []dashRecentSignal
	var firedPairs []firedSignal
	if corpus, cerr := s.buildSignalCorpus(r); cerr == nil {
		censuses := signal.EvaluateCorpus(corpus)
		for _, c := range censuses {
			sev, _ := signal.SeverityFor(c.Rule)
			for _, m := range c.Fired {
				openSignals++
				firedPairs = append(firedPairs, firedSignal{Rule: c.Rule, Subject: m.Subject})
				sevCounts[sev.String()]++
				if sev == signal.SevCritical {
					criticalSignals++
				}
			}
		}
		hasOpenSignals = true
		if insts, ierr := s.deriveSignalInstances(ctx, censuses); ierr == nil {
			for _, in := range insts {
				if len(recentSignals) == 6 {
					break
				}
				recentSignals = append(recentSignals, dashRecentSignal{
					Severity: in.Severity,
					SevLabel: sevLabel(in.Severity),
					Title:    in.Title,
					Asset:    strings.TrimSuffix(in.Asset, in.Port),
					Port:     in.Port,
					Seen:     in.Seen,
					ViewKey:  in.SigID,
				})
			}
		} else {
			log.Printf("web: dashboard: derive signal instances: %v", ierr)
		}
	} else {
		log.Printf("web: dashboard: build signal corpus: %v", cerr)
	}

	var sevBars []dashSevBar
	maxSev := 0
	for _, sv := range signal.SevOrder {
		if n := sevCounts[sv.String()]; n > maxSev {
			maxSev = n
		}
	}
	for _, sv := range signal.SevOrder {
		n := sevCounts[sv.String()]
		pct := 0
		if maxSev > 0 {
			pct = n * 100 / maxSev
		}
		sevBars = append(sevBars, dashSevBar{Sev: sv.String(), Count: n, Pct: pct})
	}

	emptyEstate := hasNames && hasServices && names == 0 && services == 0

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
		steps = firstRunChecklist(nameScopes+addrScopes, firstSeedName(seedRows), zoneUploaded, internetVantage, scanDispatched)
	}

	schedule := s.scanSchedule(ctx)
	firstRunDone := 0
	for _, st := range steps {
		if st.Done {
			firstRunDone++
		}
	}

	deltas := s.dashboardDeltas(ctx, firedPairs)

	exposed, hasExposed := s.currentExposedCount(ctx)
	certsExpiring, hasCerts := s.currentCertsExpiring(ctx)

	assetsWatched := names + services
	hasAssets := hasNames && hasServices
	statBand := []dashStat{
		{Label: "Open signals", Value: statValue(openSignals, hasOpenSignals), Caption: "firing across your estate",
			Live: len(active) > 0, HasDelta: deltas.Known, Change: deltas.OpenSignals.Change(), Tone: statTone(deltas.OpenSignals.Change(), true)},
		{Label: "Critical", Value: statValue(criticalSignals, hasOpenSignals), Caption: "highest severity",
			HasDelta: deltas.Known, Change: deltas.Critical.Change(), Tone: statTone(deltas.Critical.Change(), true)},
		{Label: "Assets watched", Value: statValue(assetsWatched, hasAssets),
			Caption:  fmt.Sprintf("%d %s · %d %s", nameScopes, plural(nameScopes, "domain", "domains"), addrScopes, plural(addrScopes, "range", "ranges")),
			HasDelta: deltas.Known, Change: deltas.AssetsWatched.Change(), Tone: "neutral"},
		{Label: "Exposed services", Value: statValue(exposed, hasExposed), Caption: "reachable from the internet",
			HasDelta: deltas.Known, Change: deltas.Exposed.Change(), Tone: statTone(deltas.Exposed.Change(), true)},
		{Label: "Certs expiring ≤30d", Value: statValue(certsExpiring, hasCerts), Caption: "expiring within 30 days",
			HasDelta: deltas.Known, Change: deltas.CertsExpiring.Change(), Tone: statTone(deltas.CertsExpiring.Change(), true)},
	}

	var coverageMeters []coverageMeterView
	if hasScopes {
		var zones []db.ListZoneDeclarationsRow
		if z, zerr := s.store.ListZoneDeclarations(ctx); zerr == nil {
			zones = z
		}
		// A nil shared-edge map keeps #989's contradiction row on /coverage, where the remedy is.
		coverageMeters = apertureMeters(seedRows, zones, walked, s.now(), nil)
	}
	var silentZone *dashSilentZone
	if len(unavailable) > 0 {
		silentZone = &dashSilentZone{Text: "position " + unavailable[0] + " went silent"}
	}

	scanDetail := ""
	if len(active) > 0 {
		if rows, derr := s.store.ListDispatchProgress(ctx, scansHistoryLimit); derr == nil {
			queued := 0
			for _, row := range rows {
				if dv := toDispatchView(row); dv.Active {
					queued += int(dv.InFlight)
				}
			}
			scanDetail = fmt.Sprintf("%d %s queued", queued, plural(queued, "subject", "subjects"))
		} else {
			log.Printf("web: dashboard: scan detail: list dispatches: %v", derr)
		}
	}

	probeDismissed := r.URL.Query().Get("probe") == "dismissed"

	data := map[string]any{
		"Title": "Dashboard", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "dashboard",
		"Scanning":  len(active) > 0,

		"EmptyEstate":   emptyEstate,
		"FirstRunSteps": steps,
		"FirstRunDone":  firstRunDone,

		"Vantages":    vantages,
		"Unavailable": unavailable,

		"StatBand":      statBand,
		"SevBars":       sevBars,
		"HasSignals":    hasOpenSignals,
		"RecentSignals": recentSignals,

		"CoverageMeters": coverageMeters,
		"SilentZone":     silentZone,

		"ScanDetail":     scanDetail,
		"ScanSchedule":   schedule,
		"ProbeDismissed": probeDismissed,

		"Deltas":    deltas,
		"HasDeltas": deltas.Known,
	}
	if hasOpenSignals && openSignals > 0 {
		data["SignalCount"] = openSignals
	}
	return data
}

func statValue(n int, ok bool) string {
	if !ok {
		return "—"
	}
	return commaInt(n)
}

func commaInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

func statTone(change int, riseIsBad bool) string {
	switch {
	case change == 0:
		return "neutral"
	case (change > 0) == riseIsBad:
		return "bad"
	default:
		return "good"
	}
}

func firstRunChecklist(scopes int, seedName string, zoneUploaded, internetVantage, scanDispatched bool) []firstRunStep {
	scopeDetail := "A seed is a boundary, not a starting gun"
	if scopes > 0 {
		lead := seedName
		if lead == "" {
			unit := "scopes"
			if scopes == 1 {
				unit = "scope"
			}
			lead = fmt.Sprintf("%d %s", scopes, unit)
		}
		scopeDetail = lead + " declared · a seed is a boundary, not a starting gun"
	}
	return []firstRunStep{
		{
			Num: 1, Done: scopes > 0,
			Title: "Declare your domain", Detail: scopeDetail,
			HasAction: scopes == 0, ActionLabel: "Declare scope", ActionHref: "/scope",
		},
		{
			Num: 2, Done: zoneUploaded,
			Title:     "Upload a zone file",
			Detail:    "Enables removal detection — you stopped telling us becomes detectable",
			HasAction: !zoneUploaded, ActionLabel: "Upload zone", ActionHref: "/scope",
		},
		{
			Num: 3, Done: internetVantage,
			Title:     "Add an internet vantage",
			Detail:    "Exposure needs an outside observer, unconditionally",
			HasAction: !internetVantage, ActionLabel: "Provision prober", ActionHref: "/settings/vantages",
		},
		{
			Num: 4, Done: scanDispatched,
			Title:     "Run the first batch",
			Detail:    "Scans dispatch on cadence; kick the first one now",
			HasAction: !scanDispatched, ActionLabel: "Run first batch", ActionPost: "/onboarding/finish",
			Gated: !internetVantage, GateTitle: "Needs an internet vantage first",
		},
	}
}

func firstSeedName(rows []db.ListSeedsRow) string {
	for _, sd := range rows {
		if sd.NameDomain.Valid && sd.NameDomain.String != "" {
			return sd.NameDomain.String
		}
		if sd.AddressCidr != nil {
			return sd.AddressCidr.String()
		}
	}
	return ""
}

func (s *server) accountPage(w http.ResponseWriter, r *http.Request, _ db.Account) {
	http.Redirect(w, r, "/settings?tab=team", http.StatusSeeOther)
}

func (s *server) createAccount(w http.ResponseWriter, r *http.Request, _ db.Account) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	role := r.FormValue("role")

	if role != roleAdmin && role != roleViewer {
		s.renderFormError(w, r, "Role must be admin or viewer.")
		return
	}
	if msg := validateCredentials(username, password); msg != "" {
		s.renderFormError(w, r, msg)
		return
	}
	if _, err := s.createAccountRow(r, username, role, password); err != nil {
		s.renderFormError(w, r, createError(err))
		return
	}
	stashFormFlash(s, r, settingsForms{flashTab: "team", notice: "Account " + username + " created."})
	s.backToSection(w, r, "team")
}

func (s *server) totpEnable(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.beginTOTPEnroll(w, r, acct)
}

func (s *server) totpEnrollForm(w http.ResponseWriter, r *http.Request, acct db.Account) {
	s.beginTOTPEnroll(w, r, acct)
}

func (s *server) beginTOTPEnroll(w http.ResponseWriter, r *http.Request, acct db.Account) {
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		s.serverError(w, "generate totp secret", err)
		return
	}
	if s.devMode {
		if err := s.devResetTOTPEnroll(r.Context(), acct.ID); err != nil {
			s.serverError(w, "dev: reset totp enrolment", err)
			return
		}
		secret = devFixtureEnrollSecret
	} else if acct.TotpEnabled {
		// Re-rolling would strip the second factor until a fresh confirm, so a stolen session cannot.
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	// The sealing key never enters Postgres, so a table leak discloses ciphertext (ADR-0053).
	enc, err := auth.EncryptTOTPSecret(s.totpKey, secret)
	if err != nil {
		s.serverError(w, "encrypt totp secret", err)
		return
	}
	if err := s.store.SetTOTPSecret(r.Context(), db.SetTOTPSecretParams{
		ID: acct.ID, TotpSecret: pgtype.Text{String: enc, Valid: true},
	}); err != nil {
		s.serverError(w, "store totp secret", err)
		return
	}
	s.render(w, r, "totp-enroll", s.signinData(totpEnrollData(acct.Username, secret, "")))
}

func totpEnrollData(username, secret, errMsg string) map[string]any {
	// The QR is built in-process, so the seed never reaches a third party (ADR-0053).
	uri := auth.OtpauthURI(secret, username, issuer)
	data := map[string]any{
		"Title": "Two-factor", "Secret": secret, "OtpauthURI": uri,
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if svg, err := qr.SVG([]byte(uri), "Two-factor enrollment QR code for "+username); err == nil {
		data["OtpauthQR"] = template.HTML(svg) // #nosec G203 (SVG built by internal qr encoder; username html.EscapeString-escaped into aria-label)
	}
	return data
}

func (s *server) totpConfirm(w http.ResponseWriter, r *http.Request, acct db.Account) {
	fresh, err := s.store.GetAccountByID(r.Context(), acct.ID)
	if err != nil || !fresh.TotpSecret.Valid {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !(s.devMode && r.FormValue("code") == devFixtureTOTPCode) {
		secret, derr := auth.DecryptTOTPSecret(s.totpKey, fresh.TotpSecret.String)
		if derr != nil {
			s.serverError(w, "decrypt totp secret", derr)
			return
		}
		if !auth.VerifyTOTP(secret, r.FormValue("code"), s.now()) {
			s.render(w, r, "totp-enroll", s.signinData(totpEnrollData(acct.Username, secret,
				"Incorrect code. Two-factor is not enabled.")))
			return
		}
	}
	if err := s.store.ConfirmTOTP(r.Context(), acct.ID); err != nil {
		s.serverError(w, "confirm totp", err)
		return
	}
	fresh.TotpEnabled = true

	// A silent skip here would leave two-factor on with no recovery path, so it is fatal.
	plain, hashes, err := s.recoveryCodes()
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
	s.render(w, r, "totp-recovery", s.signinData(map[string]any{"Title": "Two-factor", "Codes": plain}))
}

func (s *server) forgotForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "forgot", s.signinData(map[string]any{"Title": "Reset password"}))
}

func (s *server) forgotSubmit(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	if acct, err := s.store.GetAccountByUsername(r.Context(), username); err == nil {
		if plaintext, hash, terr := newOpaqueToken(); terr != nil {
			log.Printf("web: forgot: mint reset token: %v", terr)
		} else if pr, cerr := s.store.CreatePasswordReset(r.Context(), db.CreatePasswordResetParams{
			AccountID: acct.ID, TokenHash: hash,
			ExpiresAt: pgtype.Timestamptz{Time: s.now().Add(s.resetTTL), Valid: true},
		}); cerr != nil {
			log.Printf("web: forgot: create reset: %v", cerr)
		} else {
			// The plaintext resets the account, so it must never land in a log by default (CWE-532, #328).
			log.Printf("web: password reset requested for %q (reset id %d, expires in %s)", // #nosec G706 (sanitized via logSafe)
				logSafe(username), pr.ID, s.resetTTL)
			// A self-hosted host has no mail, so the link is opt-in for the operator's own logs.
			if env.OrDefault("VERGE_LOG_RESET_LINKS", "") != "" {
				log.Printf("web: password reset link for %q: /reset?token=%s", logSafe(username), plaintext) // #nosec G706 (sanitized via logSafe)
			}
		}
	}
	// The response is identical for an unknown account, so the endpoint enumerates nothing.
	s.render(w, r, "forgot-sent", s.signinData(map[string]any{"Title": "Reset password"}))
}

func (s *server) resetForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if _, ok := s.lookupReset(r, token); !ok {
		s.render(w, r, "reset-invalid", s.signinData(map[string]any{"Title": "Reset password"}))
		return
	}
	s.render(w, r, "reset", s.signinData(map[string]any{"Title": "Set a new password", "Token": token}))
}

func (s *server) resetSubmit(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	pr, ok := s.lookupReset(r, token)
	if !ok {
		s.render(w, r, "reset-invalid", s.signinData(map[string]any{"Title": "Reset password"}))
		return
	}
	pw := r.FormValue("password")
	confirm := r.FormValue("confirm")
	fail := func(msg string) {
		s.renderStatus(w, r, http.StatusBadRequest, "reset", s.signinData(map[string]any{"Title": "Set a new password", "Token": token, "Error": msg}))
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
	// A reset presumes the old password is lost, so every session goes with no exception (ADR-0117).
	if err := s.store.RevokeAllSessionsForAccount(r.Context(), db.RevokeAllSessionsForAccountParams{
		AccountID: pr.AccountID,
		RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		log.Printf("web: reset: revoke all sessions: %v", err)
	}
	s.render(w, r, "reset-done", s.signinData(map[string]any{"Title": "Password updated"}))
}

func (s *server) lookupReset(r *http.Request, token string) (db.PasswordReset, bool) {
	// The expiry check is here, not in SQL, so a fixed-clock test and production agree.
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

func (s *server) inviteForm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	inv, ok := s.lookupInvite(r, token)
	if !ok {
		s.render(w, r, "invite-invalid", s.signinData(map[string]any{"Title": "Invitation"}))
		return
	}
	s.render(w, r, "invite", s.signinData(map[string]any{"Title": "Accept invitation", "Token": token, "Role": inv.Role}))
}

func (s *server) inviteAccept(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	inv, ok := s.lookupInvite(r, token)
	if !ok {
		s.render(w, r, "invite-invalid", s.signinData(map[string]any{"Title": "Invitation"}))
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	fail := func(msg string) {
		s.renderStatus(w, r, http.StatusBadRequest, "invite", s.signinData(map[string]any{
			"Title": "Accept invitation", "Token": token, "Role": inv.Role,
			"Error": msg, "Username": username,
		}))
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
	// No session is minted here, so a bare invite token never yields privileged state.
	http.Redirect(w, r, "/login?invited=1", http.StatusSeeOther)
}

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

const recoveryCodeCount = 8

// The visually ambiguous characters are out, so a code read off a screen transcribes cleanly.

const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// 28 characters over a 31-symbol alphabet clear the 128-bit bar a fallback credential needs (#338).

const recoveryCodeChars = 28

const recoveryGroupSize = 4

func newOpaqueToken() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plaintext = base64.RawURLEncoding.EncodeToString(b)
	return plaintext, hashToken(plaintext), nil
}

func hashToken(plaintext string) string {
	// These tokens are 256-bit random, so a digest suffices where a password would need a slow KDF.
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func newRecoveryCodes(n int) (plain, hashes []string, err error) {
	// A per-code bcrypt hash, salted and slow, keeps a leaked dump from offline cracking (#338).
	plain = make([]string, 0, n)
	hashes = make([]string, 0, n)
	for i := 0; i < n; i++ {
		code, cerr := newRecoveryCode()
		if cerr != nil {
			return nil, nil, cerr
		}
		h, herr := auth.HashPassword(code)
		if herr != nil {
			return nil, nil, herr
		}
		plain = append(plain, code)
		hashes = append(hashes, h)
	}
	return plain, hashes, nil
}

func newRecoveryCode() (string, error) {
	const max = 256 - (256 % len(recoveryAlphabet))
	var sb strings.Builder
	buf := make([]byte, 1)
	for i := 0; i < recoveryCodeChars; i++ {
		if i > 0 && i%recoveryGroupSize == 0 {
			sb.WriteByte('-')
		}
		for {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			// Rejecting the tail byte removes the modulo bias a bare remainder draw would carry (#338).
			if int(buf[0]) < max {
				sb.WriteByte(recoveryAlphabet[int(buf[0])%len(recoveryAlphabet)])
				break
			}
		}
	}
	return sb.String(), nil
}

func normalizeRecoveryCode(s string) string {
	// The stored hash is of the dashed form, so the dashes must survive normalisation.
	return strings.ToLower(strings.TrimSpace(s))
}

func subtleConstantEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/profile.tmpl"))

type profileTokenView struct {
	ID      int64
	Name    string
	Prefix  string
	Created string
	Last    string
}

type profileSessionView struct {
	ID         int64
	Device     string
	IP         string
	LastActive string
	Current    bool
}

type profileState struct {
	notice        string
	pwError       string
	createOpen    bool
	tokError      string
	tokName       string
	minted        string
	mintedName    string
	revokeID      int64
	revokeErr     string
	endSession    bool
	signOutOthers bool
	ssoNotice     string
	ssoError      string
}

func (s *server) profilePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	st, _ := takeFormFlash[profileState](s, r)
	q := r.URL.Query()
	if q.Get("new") != "" {
		st.createOpen = true
	}
	if v := q.Get("revoke"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			st.revokeID = id
		} else if s.devMode {
			st.revokeID = s.devResolveFixtureTokenID(r, acct.ID, v)
		}
	}
	if q.Get("endsession") != "" {
		st.endSession = true
	}
	if q.Get("signoutothers") != "" {
		st.signOutOthers = true
	}
	// The outcome is a fixed code mapped here, so no free text from the URL is ever reflected.
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

const profilePath = "/profile"

func (s *server) failProfile(w http.ResponseWriter, r *http.Request, st profileState) {
	// The bare path, not the submitting URL: returning would re-open a spent confirm (ADR-0130 §3).
	stashFormFlash(s, r, st)
	http.Redirect(w, r, profilePath, http.StatusSeeOther)
}

func (s *server) renderProfile(w http.ResponseWriter, r *http.Request, acct db.Account, st profileState) {
	// The passed account is a stale copy; TOTP enrolment in this session mutates the row.
	if fresh, err := s.store.GetAccountByID(r.Context(), acct.ID); err == nil {
		acct = fresh
	}

	var tokens []profileTokenView
	if rows, err := s.listPersonalTokensCreatedAsc(r.Context(), acct.ID); err == nil {
		for _, t := range rows {
			tokens = append(tokens, profileTokenView{
				ID: t.ID, Name: t.Name, Prefix: t.Prefix,
				Created: isoDate(t.CreatedAt), Last: lastUsed(t.LastUsedAt, s.now()),
			})
		}
	} else {
		log.Printf("web: profile: list personal tokens: %v", err)
	}

	// A failed read suppresses the offer too, so a blip never invites re-linking a provider.
	linked, linkedProviders, ok := s.profileSSOIdentities(r, acct.ID)
	var available []profileLinkView
	if ok {
		available = s.profileLinkableProviders(r, linkedProviders)
	}

	curSessionID, haveCurSession := s.currentSessionID(r)
	var sessions []profileSessionView
	if rows, err := s.store.ListSessionsForAccount(r.Context(), db.ListSessionsForAccountParams{
		AccountID: acct.ID,
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err == nil {
		for _, row := range rows {
			sessions = append(sessions, profileSessionView{
				ID:         row.ID,
				Device:     sessionDeviceFromUA(row.UserAgent),
				IP:         row.Ip,
				LastActive: profileRelTime(row.LastSeenAt.Time, s.now()),
				Current:    haveCurSession && row.ID == curSessionID,
			})
		}
	} else {
		log.Printf("web: profile: list sessions: %v", err)
	}

	apiEnabled := false
	if cfg, err := s.store.GetInstanceConfig(r.Context()); err == nil {
		apiEnabled = cfg.ApiEnabled
	} else {
		log.Printf("web: profile: instance config: %v", err)
	}

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

		"Sessions": sessions,

		"Tokens":     tokens,
		"APIEnabled": apiEnabled,

		"SSOIdentities": linked,
		"SSOProviders":  available,
		"SSONotice":     st.ssoNotice,
		"SSOError":      st.ssoError,

		"Notice":        st.notice,
		"PwError":       st.pwError,
		"CreateOpen":    st.createOpen,
		"TokError":      st.tokError,
		"TokName":       st.tokName,
		"Minted":        st.minted,
		"MintedName":    st.mintedName,
		"RevokeID":      st.revokeID,
		"RevokeName":    revokeName,
		"RevokeErr":     st.revokeErr,
		"EndSession":    st.endSession,
		"SignOutOthers": st.signOutOthers,
	}
	s.render(w, r, "profile", data)
}

func (s *server) listPersonalTokensCreatedAsc(ctx context.Context, accountID int64) ([]db.ListPersonalTokensRow, error) {
	rows, err := s.store.ListPersonalTokens(ctx, accountID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		ti, tj := rows[i].CreatedAt.Time, rows[j].CreatedAt.Time
		if ti.Equal(tj) {
			return rows[i].ID < rows[j].ID
		}
		return ti.Before(tj)
	})
	return rows, nil
}

func (s *server) devResolveFixtureTokenID(r *http.Request, accountID int64, ref string) int64 {
	if !strings.HasPrefix(ref, "t") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(ref, "t"))
	if err != nil || n < 1 {
		return 0
	}
	rows, err := s.listPersonalTokensCreatedAsc(r.Context(), accountID)
	if err != nil || n > len(rows) {
		return 0
	}
	return rows[n-1].ID
}

type profileIdentityView struct {
	ID          int64
	Provider    string
	DisplayName string
	LinkedAt    string
}

type profileLinkView struct {
	Slug string
	Name string
}

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

func (s *server) changePassword(w http.ResponseWriter, r *http.Request, acct db.Account) {
	current := r.FormValue("current_password")
	next := r.FormValue("new_password")

	fresh, err := s.store.GetAccountByID(r.Context(), acct.ID)
	if err != nil {
		s.serverError(w, "profile: read account", err)
		return
	}
	if !auth.CheckPassword(fresh.PasswordHash, current) {
		s.failProfile(w, r, profileState{pwError: "Current password is incorrect."})
		return
	}
	if msg := validatePassword(next); msg != "" {
		s.failProfile(w, r, profileState{pwError: msg})
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
	// A changed password must kill every other session, so a stolen old one is dead (ADR-0117, #408).
	if curID, ok := s.currentSessionID(r); ok {
		if err := s.store.RevokeOtherSessionsForAccount(r.Context(), db.RevokeOtherSessionsForAccountParams{
			AccountID: acct.ID,
			ID:        curID,
			RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
		}); err != nil {
			log.Printf("web: profile: revoke other sessions after password change: %v", err)
		}
	} else {
		// Revoking with no exception would sign the caller out of the tab they just changed.
		log.Printf("web: profile: password changed but current session id did not resolve; other sessions left in place")
	}
	s.toastRedirect(w, r, profilePath, "ok", "Password changed", "Other sessions keep working until they expire.")
}

func (s *server) createPersonalToken(w http.ResponseWriter, r *http.Request, acct db.Account) {
	name := strings.TrimSpace(r.FormValue("name"))
	switch {
	case name == "":
		s.failProfile(w, r, profileState{createOpen: true, tokError: "Give the token a name.", tokName: name})
		return
	case len(name) > 64:
		s.failProfile(w, r, profileState{createOpen: true, tokError: "Name must be 64 characters or fewer.", tokName: name})
		return
	}
	plaintext, prefix, hash, err := s.newPersonalToken()
	if err != nil {
		s.serverError(w, "profile: mint token", err)
		return
	}
	if _, err := s.store.CreatePersonalToken(r.Context(), db.CreatePersonalTokenParams{
		AccountID: acct.ID, Name: name, Prefix: prefix, TokenHash: hash,
	}); err != nil {
		if isUniqueViolation(err) {
			s.failProfile(w, r, profileState{createOpen: true, tokError: "You already have a token named that.", tokName: name})
			return
		}
		s.serverError(w, "profile: create token", err)
		return
	}
	s.revealMintedToken(w, r, acct, plaintext, name)
}

func (s *server) revealMintedToken(w http.ResponseWriter, r *http.Request, acct db.Account, plaintext, name string) {
	// Its own method so the console-answer guard exempts this answer, not the whole route.
	// A redirect could not carry the plaintext without stashing key material to survive it.
	s.renderProfile(w, r, acct, profileState{minted: plaintext, mintedName: name})
}

func (s *server) revokePersonalToken(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)

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
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	if err := s.store.DeletePersonalToken(r.Context(), db.DeletePersonalTokenParams{ID: id, AccountID: acct.ID}); err != nil {
		s.serverError(w, "profile: revoke token", err)
		return
	}
	s.toastRedirect(w, r, "/profile", "neutral", "Token revoked", name)
}

func (s *server) revokeSession(w http.ResponseWriter, r *http.Request, _ db.Account) {
	s.revokeCurrentSession(r)
	s.clearCookie(w, sessionCookie)
	s.toastRedirect(w, r, "/login", "neutral", "Session ended", "Signed out on this device.")
}

func (s *server) revokeOneSession(w http.ResponseWriter, r *http.Request, acct db.Account) {
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	curID, haveCur := s.currentSessionID(r)
	device := ""
	if sess, err := s.store.ListSessionsForAccount(r.Context(), db.ListSessionsForAccountParams{
		AccountID: acct.ID,
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err == nil {
		for _, row := range sess {
			if row.ID == id {
				device = sessionDeviceFromUA(row.UserAgent)
				break
			}
		}
	}
	// The account_id predicate is what stops a posted id ending another account's session.
	if err := s.store.RevokeSession(r.Context(), db.RevokeSessionParams{
		ID:        id,
		AccountID: acct.ID,
		RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "profile: revoke session", err)
		return
	}
	if haveCur && id == curID {
		s.clearCookie(w, sessionCookie)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.toastRedirect(w, r, "/profile", "neutral", "Session revoked", device+" signs out on its next request.")
}

func (s *server) signOutOtherSessions(w http.ResponseWriter, r *http.Request, acct db.Account) {
	curID, ok := s.currentSessionID(r)
	if !ok {
		http.Redirect(w, r, "/profile", http.StatusSeeOther)
		return
	}
	ended := 0
	if sess, err := s.store.ListSessionsForAccount(r.Context(), db.ListSessionsForAccountParams{
		AccountID: acct.ID,
		ExpiresAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err == nil {
		for _, row := range sess {
			if row.ID != curID {
				ended++
			}
		}
	}
	if err := s.store.RevokeOtherSessionsForAccount(r.Context(), db.RevokeOtherSessionsForAccountParams{
		AccountID: acct.ID,
		ID:        curID,
		RevokedAt: pgtype.Timestamptz{Time: s.now(), Valid: true},
	}); err != nil {
		s.serverError(w, "profile: sign out other sessions", err)
		return
	}
	s.toastRedirect(w, r, "/profile", "neutral", "Other sessions signed out", strconv.Itoa(ended)+" sessions ended.")
}

func (s *server) newPersonalToken() (plaintext, prefix, hash string, err error) {
	if s.devMode {
		return fixtureMintedToken()
	}
	return mintPersonalToken()
}

func fixtureMintedToken() (plaintext, prefix, hash string, err error) {
	plaintext = devFixtureMintedToken
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	prefix = plaintext[:11] + "…"
	return plaintext, prefix, hash, nil
}

func mintPersonalToken() (plaintext, prefix, hash string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return "", "", "", err
	}
	plaintext = "vg_pat_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(plaintext))
	hash = hex.EncodeToString(sum[:])
	prefix = plaintext[:11] + "…"
	return plaintext, prefix, hash, nil
}

func validatePassword(pw string) string {
	// bcrypt hashes at most 72 bytes and errors beyond, so the cap is the library's, not ours.
	switch {
	case len(pw) < 12:
		return "Password must be at least 12 characters."
	case len(pw) > 72:
		return "Password must be 72 characters or fewer."
	default:
		return ""
	}
}

func isoDate(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return "—"
	}
	return ts.Time.Format("2006-01-02")
}

func lastUsed(ts pgtype.Timestamptz, now time.Time) string {
	// The frozen profile.tmpl renders its own "never" branch, so an empty string, not a dash.
	if !ts.Valid {
		return ""
	}
	return profileRelTime(ts.Time, now)
}

func profileRelTime(t, now time.Time) string {
	// Deliberately unlike messages.go relTime: the Profile counts days past 7d, never weeks.
	d := now.Sub(t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	default:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d"
	}
}

func initials(username string) string {
	fields := strings.FieldsFunc(username, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || unicode.IsSpace(r)
	})
	switch len(fields) {
	case 0:
		return "?"
	case 1:
		r := []rune(strings.ToUpper(fields[0]))
		if len(r) == 1 {
			return string(r[0])
		}
		return string(r[0]) + string(r[1])
	default:
		a := []rune(strings.ToUpper(fields[0]))
		b := []rune(strings.ToUpper(fields[1]))
		return string(a[0]) + string(b[0])
	}
}

func sessionIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *server) createAccountRow(r *http.Request, username, role, password string) (db.Account, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return db.Account{}, err
	}
	return s.store.CreateAccount(r.Context(), db.CreateAccountParams{
		Username: username, Role: role, PasswordHash: hash,
	})
}

func validateCredentials(username, password string) string {
	switch {
	case username == "":
		return "Username is required."
	case len(username) > 64:
		return "Username must be 64 characters or fewer."
	case len(password) < 12:
		return "Password must be at least 12 characters."
	case len(password) > 72:
		return "Password must be 72 characters or fewer."
	default:
		return ""
	}
}

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

func isForeignKeyViolation(err error) bool {
	var e interface{ SQLState() string }
	return errors.As(err, &e) && e.SQLState() == "23503"
}

func (s *server) renderFormError(w http.ResponseWriter, r *http.Request, msg string) {
	s.failSettings(w, r, settingsForms{section: "team", teamError: msg})
}

func (s *server) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	s.renderStatus(w, r, http.StatusOK, name, data)
}

func (s *server) renderStatus(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	s.injectChrome(data, r)
	// WriteHeader commits the header block, so a Content-Type set after it is silently dropped.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("web: render %s: %v", name, err)
	}
}

func (s *server) injectChrome(data any, r *http.Request) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}
	if _, isChrome := m["IsAdmin"]; !isChrome {
		return
	}
	navActive, _ := m["NavActive"].(string)
	scanning, hasScanning := m["Scanning"].(bool)

	if _, set := m["BackURL"]; !set {
		m["BackURL"] = backURL(r)
	}

	if s.devMode {
		showToast := r != nil && r.URL.Query().Get("variant") == devShellToastVariant
		m["Chrome"] = chromeFromFixture(navActive, scanning, showToast)
		return
	}

	ctx := context.Background()
	acct, hasAcct := m["Account"].(db.Account)
	signalCount, _ := m["SignalCount"].(int)

	if !hasScanning {
		scanning = s.chromeScanRunning(ctx)
	}

	unread, hasUnread := m["Unread"].(int64)
	if !hasUnread && hasAcct {
		n, err := s.store.CountUnreadMessages(ctx, acct.ID)
		if err != nil {
			log.Printf("web: unread count: %v", err)
		}
		unread = n
	}

	var messages []bellMessage
	if hasAcct {
		messages = s.bellMessages(ctx, acct.ID, 4)
	}

	toasts := decodeToasts(r)
	initials := "?"
	userName := ""
	if hasAcct {
		initials = accountInitials(acct.Username)
		userName = acct.Username
	}

	if hasAcct {
		if t, ok := s.flash.take(acct.ID); ok {
			toasts = []toastVM{t}
		}
	}

	m["Chrome"] = &chromeVM{
		Nav:           navSlice(navActive, signalCount),
		Org:           "self-hosted",
		Version:       s.buildVersion(),
		UserName:      userName,
		UserInitials:  initials,
		ScanRunning:   scanning,
		Unread:        unread > 0,
		Messages:      messages,
		PaletteGroups: paletteGroupsProd(signalCount, unread),
		Toasts:        toasts,
	}
}

func (s *server) serverError(w http.ResponseWriter, what string, err error) {
	log.Printf("web: %s: %v", what, err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func (s *server) setSignedCookie(w http.ResponseWriter, r *http.Request, name string, kind auth.Kind, id int64, sessionToken string, ttl time.Duration) bool {
	// SameSite=Lax is the CSRF defence for every mutating endpoint; there is no token check.
	token, err := auth.SignSession(s.key, auth.Session{
		AccountID: id, Kind: kind, ExpiresAt: s.now().Add(ttl), Token: sessionToken,
	})
	if err != nil {
		s.serverError(w, "sign session", err)
		return false
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure set conditionally (r.TLS != nil || s.secureCookies) for plain-HTTP dev/proxy; HttpOnly + SameSite=Lax always set.
		Name: name, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.secureCookies || r.TLS != nil, MaxAge: int(ttl.Seconds()),
	})
	return true
}

func (s *server) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- deletion cookie (empty value, MaxAge<0); Secure via s.secureCookies expression; HttpOnly + SameSite=Lax set.
		Name: name, Value: "", Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: s.secureCookies, MaxAge: -1,
	})
}
