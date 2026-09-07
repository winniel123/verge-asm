package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/seed"
)

type healthzStore interface {
	RecordHeartbeat(ctx context.Context) (db.Heartbeat, error)
}

type addressCapStore interface {
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
}

type store interface {
	// Each handler group states its own reach, so this sums them and names no query (ADR-0149 §3).

	addressCapStore
	adminSessionStore
	annotationsStore
	apertureSettingsStore
	apiAuthStore
	apiV1Store
	backupStore
	channelSendTestStore
	channelsStore
	chromeStore
	coldStore
	custodyCensusStore
	custodyStore
	dashboardStore
	deliverySettingsStore
	deltasStore
	devFixtureStore
	driftStore
	exclusionsStore
	exposureStore
	graphStore
	healthzStore
	instanceSettingsStore
	integrationsStore
	inventoryStore
	inviteAcceptStore
	loginStore
	messagesStore
	passwordStore
	personalTokenStore
	probersStore
	profileStore
	proposalsStore
	rawOutputStore
	reportScheduleRowStore
	reportScheduleStore
	reportsExportStore
	reportsStore
	restoreStore
	scanTriggerStore
	scansStore
	searchStore
	seedsStore
	sessionStore
	shellStore
	signalsStore
	sourcesStore
	ssoAdminStore
	ssoAuthStore
	subjectsStore
	teamAdminStore
	totpEnrollStore
	vantageClassStore
	vergeCoreStore
}

type server struct {
	// A field per group binds the split at compile time (ADR-0149 §3).

	addressCapStore        addressCapStore
	adminSessionStore      adminSessionStore
	annotationsStore       annotationsStore
	apertureSettingsStore  apertureSettingsStore
	apiAuthStore           apiAuthStore
	apiV1Store             apiV1Store
	backupStore            backupStore
	channelSendTestStore   channelSendTestStore
	channelsStore          channelsStore
	chromeStore            chromeStore
	coldStore              coldStore
	custodyCensusStore     custodyCensusStore
	custodyStore           custodyStore
	dashboardStore         dashboardStore
	deliverySettingsStore  deliverySettingsStore
	deltasStore            deltasStore
	devFixtureStore        devFixtureStore
	driftStore             driftStore
	exclusionsStore        exclusionsStore
	exposureStore          exposureStore
	graphStore             graphStore
	healthzStore           healthzStore
	instanceSettingsStore  instanceSettingsStore
	integrationsStore      integrationsStore
	inventoryStore         inventoryStore
	inviteAcceptStore      inviteAcceptStore
	loginStore             loginStore
	messagesStore          messagesStore
	passwordStore          passwordStore
	personalTokenStore     personalTokenStore
	probersStore           probersStore
	profileStore           profileStore
	proposalsStore         proposalsStore
	rawOutputStore         rawOutputStore
	reportScheduleRowStore reportScheduleRowStore
	reportScheduleStore    reportScheduleStore
	reportsExportStore     reportsExportStore
	reportsStore           reportsStore
	restoreStore           restoreStore
	scanTriggerStore       scanTriggerStore
	scansStore             scansStore
	searchStore            searchStore
	seedsStore             seedsStore
	sessionStore           sessionStore
	shellStore             shellStore
	signalsStore           signalsStore
	sourcesStore           sourcesStore
	ssoAdminStore          ssoAdminStore
	ssoAuthStore           ssoAuthStore
	subjectsStore          subjectsStore
	teamAdminStore         teamAdminStore
	totpEnrollStore        totpEnrollStore
	vantageClassStore      vantageClassStore
	vergeCoreStore         vergeCoreStore

	key []byte

	// A read-only database leak must disclose ciphertext and no key (ADR-0053).

	totpKey       []byte
	transcriptKey []byte
	setupToken    string
	now           func() time.Time
	startedAt     time.Time
	sessionTTL    time.Duration
	pendingTTL    time.Duration
	resetTTL      time.Duration
	proposer      proposerRunner

	sso ssoFlow

	channelSender channelTestSender

	dispatcher scanTrigger

	secureCookies bool

	// A forwarded-for header is caller-supplied, so it keys the rate limiter, never authorization.

	trustedProxies trustedProxies

	// Host headers are attacker-controlled, so the OIDC redirect_uri never derives from one (#293).

	externalURL string

	// Two concurrent valid setups would each pass the no-accounts check and spend the token twice.

	setupMu sync.Mutex

	flash *flashStore

	formFlash *formFlashStore

	loginLimiter *loginLimiter

	progress progressEvents

	// sqlc generates no goose_db_version read, so the pool serves what internal/db cannot (#391).

	pool *pgxpool.Pool

	devMode bool

	stateDir string

	restoreMu    sync.Mutex
	restoreStage map[int64]*restoreStaging

	coverageMu        sync.Mutex
	coverageEmptyOnce bool

	routes *http.ServeMux
}

func newServer(s store, key []byte, setupToken string, now func() time.Time) *server {
	// A nil key fails closed rather than admitting cleartext to Postgres (ADR-0172 §2, #337).
	totpKey, _ := auth.DeriveTOTPKey(key)
	return &server{
		addressCapStore:        s,
		adminSessionStore:      s,
		annotationsStore:       s,
		apertureSettingsStore:  s,
		apiAuthStore:           s,
		apiV1Store:             s,
		backupStore:            s,
		channelSendTestStore:   s,
		channelsStore:          s,
		chromeStore:            s,
		coldStore:              s,
		custodyCensusStore:     s,
		custodyStore:           s,
		dashboardStore:         s,
		deliverySettingsStore:  s,
		deltasStore:            s,
		devFixtureStore:        s,
		driftStore:             s,
		exclusionsStore:        s,
		exposureStore:          s,
		graphStore:             s,
		healthzStore:           s,
		instanceSettingsStore:  s,
		integrationsStore:      s,
		inventoryStore:         s,
		inviteAcceptStore:      s,
		loginStore:             s,
		messagesStore:          s,
		passwordStore:          s,
		personalTokenStore:     s,
		probersStore:           s,
		profileStore:           s,
		proposalsStore:         s,
		rawOutputStore:         s,
		reportScheduleRowStore: s,
		reportScheduleStore:    s,
		reportsExportStore:     s,
		reportsStore:           s,
		restoreStore:           s,
		scanTriggerStore:       s,
		scansStore:             s,
		searchStore:            s,
		seedsStore:             s,
		sessionStore:           s,
		shellStore:             s,
		signalsStore:           s,
		sourcesStore:           s,
		ssoAdminStore:          s,
		ssoAuthStore:           s,
		subjectsStore:          s,
		teamAdminStore:         s,
		totpEnrollStore:        s,
		vantageClassStore:      s,
		vergeCoreStore:         s,
		key:                    key,
		totpKey:                totpKey,
		setupToken:             setupToken,
		now:                    now,
		startedAt:              now(),
		sessionTTL:             12 * time.Hour,
		pendingTTL:             5 * time.Minute,
		resetTTL:               30 * time.Minute,
		proposer:               proposer.DefaultRegistry(newOutboundClient(30 * time.Second)),
		sso:                    newOIDCFlow(newOutboundClient(30 * time.Second)),
		channelSender:          newHTTPChannelSender(now),
		loginLimiter:           newLoginLimiter(now),
		flash:                  newFlashStore(),
		formFlash:              newFormFlashStore(),
		restoreStage:           make(map[int64]*restoreStaging),
	}
}

func (s *server) obsAsOf() pgtype.Timestamptz {
	// An evidential row is discardable at any age, so a derivation reads the live tier (ADR-0041).
	return pgtype.Timestamptz{Time: s.now().UTC(), Valid: true}
}

func (s *server) addressCap(ctx context.Context) int {
	// The cap is read at declaration only, so lowering it invalidates no declared scope (ADR-0127).
	cfg, err := s.addressCapStore.GetInstanceConfig(ctx)
	if err != nil || cfg.SeedAddressCap <= 0 {
		return seed.DefaultAddressCap
	}
	return int(cfg.SeedAddressCap)
}

func (s *server) redirectTo(target string, code int) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, _ db.Account) {
		// A moved route keeps its login gate, so an unauthenticated hit lands on /login (#286).
		dst := target
		if r.URL.RawQuery != "" {
			sep := "?"
			if strings.Contains(target, "?") {
				sep = "&"
			}
			dst += sep + r.URL.RawQuery
		}
		http.Redirect(w, r, dst, code) // #nosec G710 (target is a constant internal route; RawQuery appended only as its query string, no host control)
	}
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /", s.requireLogin(s.home))

	mux.HandleFunc("GET /setup", s.setupForm)
	mux.HandleFunc("POST /setup", s.setupSubmit)

	mux.HandleFunc("GET /login", s.loginForm)
	mux.HandleFunc("POST /login", s.loginSubmit)
	mux.HandleFunc("POST /login/totp", s.loginTOTP)
	mux.HandleFunc("POST /logout", s.logout)
	mux.HandleFunc("POST /signout", s.logout)

	mux.HandleFunc("GET /login/sso/{slug}", s.ssoStart)
	mux.HandleFunc("GET /login/sso/{slug}/callback", s.ssoCallback)

	mux.HandleFunc("GET /forgot", s.forgotForm)
	mux.HandleFunc("POST /forgot", s.forgotSubmit)
	mux.HandleFunc("GET /reset", s.resetForm)
	mux.HandleFunc("POST /reset", s.resetSubmit)
	mux.HandleFunc("GET /invite", s.inviteForm)
	mux.HandleFunc("POST /invite", s.inviteAccept)

	mux.HandleFunc("GET /onboarding", s.requireLogin(s.onboarding))
	mux.HandleFunc("POST /onboarding", s.requireLogin(s.onboardingStep))
	mux.HandleFunc("POST /onboarding/finish", s.requireAdmin(s.finishOnboarding))

	mux.HandleFunc("GET /scope", s.requireLogin(s.seedsPage))
	mux.HandleFunc("GET /seeds", s.requireLogin(s.redirectTo("/scope", http.StatusMovedPermanently)))
	mux.HandleFunc("POST /seeds", s.requireAdmin(s.declareSeed))
	mux.HandleFunc("POST /seeds/preview", s.requireAdmin(s.previewSeedWithdrawal))
	mux.HandleFunc("POST /seeds/delete", s.requireAdmin(s.deleteSeed))
	mux.HandleFunc("POST /seeds/custody", s.requireAdmin(s.setCustody))
	mux.HandleFunc("POST /seeds/zone", s.requireAdmin(s.uploadZoneFile))
	mux.HandleFunc("POST /seeds/zone/interval", s.requireAdmin(s.setZoneInterval))
	mux.HandleFunc("POST /exclusions", s.requireAdmin(s.declareExclusion))
	mux.HandleFunc("POST /exclusions/preview", s.requireAdmin(s.previewExclusion))
	mux.HandleFunc("POST /exclusions/delete", s.requireAdmin(s.unexclude))
	mux.HandleFunc("POST /settings/cold", s.requireAdmin(s.setColdScope))
	mux.HandleFunc("POST /settings/probers", s.requireAdmin(s.provisionProber))
	mux.HandleFunc("POST /settings/vantages/resolver", s.requireAdmin(s.setVantageResolver))

	mux.HandleFunc("GET /exposure", s.requireLogin(s.exposurePage))
	mux.HandleFunc("GET /settings/vantages", s.requireLogin(s.redirectTo("/settings?tab=vantages", http.StatusSeeOther)))
	mux.HandleFunc("GET /reports", s.requireLogin(s.reportsPage))
	mux.HandleFunc("GET /reports/export", s.requireLogin(s.reportsExport))
	mux.HandleFunc("GET /reports/delivery", s.requireLogin(s.reportDeliveryPage))
	mux.HandleFunc("GET /reports/delivery/pdf", s.requireLogin(s.reportDeliveryPDF))
	mux.HandleFunc("GET /reports/schedule/new", s.requireAdmin(s.newReportScheduleWizard))
	mux.HandleFunc("POST /reports/schedule/new", s.requireAdmin(s.createReportSchedule))
	mux.HandleFunc("GET /reports/schedule/{id}/edit", s.requireAdmin(s.editReportScheduleWizard))
	mux.HandleFunc("POST /reports/schedule/{id}/edit", s.requireAdmin(s.editReportSchedule))
	mux.HandleFunc("POST /reports/schedule/run", s.requireAdmin(s.runReportScheduleNow))
	mux.HandleFunc("POST /reports/schedule/delete", s.requireAdmin(s.deleteReportSchedule))

	mux.HandleFunc("GET /subjects", s.requireLogin(s.redirectTo("/inventory", http.StatusMovedPermanently)))
	// A Service or Endpoint key carries / and @, so those two read their key from a query.
	mux.HandleFunc("GET /subjects/service", s.requireLogin(s.servicePage))
	mux.HandleFunc("GET /subjects/endpoint", s.requireLogin(s.endpointPage))
	mux.HandleFunc("GET /subjects/{key}", s.requireLogin(s.subjectPage))

	mux.HandleFunc("GET /inventory", s.requireLogin(s.inventoryPage))
	mux.HandleFunc("GET /inventory/export", s.requireLogin(s.inventoryExport))
	mux.HandleFunc("GET /asset/{key}", s.requireLogin(s.assetPage))
	mux.HandleFunc("GET /drift", s.requireLogin(s.driftPage))
	mux.HandleFunc("GET /drift/export", s.requireLogin(s.driftExport))
	mux.HandleFunc("GET /run/{id}", s.requireLogin(s.runPage))
	mux.HandleFunc("GET /runs/{id}", s.requireLogin(s.runPage))
	mux.HandleFunc("GET /run/{id}/stream", s.requireLogin(s.runStream))
	mux.HandleFunc("GET /runs/{id}/stream", s.requireLogin(s.runStream))
	// Raw output can carry secrets the redacted log cannot (raw-job-output §5.2), so admin-only.
	mux.HandleFunc("GET /run/{id}/raw", s.requireAdmin(s.rawOutputPage))
	mux.HandleFunc("GET /runs/{id}/raw", s.requireAdmin(s.rawOutputPage))

	mux.HandleFunc("GET /signals", s.requireLogin(s.signalsPage))
	mux.HandleFunc("GET /signals/export", s.requireLogin(s.signalsExport))
	mux.HandleFunc("POST /annotations", s.requireAdmin(s.declareAnnotation))
	mux.HandleFunc("POST /annotations/withdraw", s.requireAdmin(s.withdrawAnnotation))

	mux.HandleFunc("GET /graph", s.requireLogin(s.graphPage))

	mux.HandleFunc("GET /search", s.requireLogin(s.searchPage))

	mux.HandleFunc("POST /proposals", s.requireAdmin(s.runLookup))
	mux.HandleFunc("POST /proposals/search", s.requireAdmin(s.runLookup))
	mux.HandleFunc("POST /proposals/confirm", s.requireAdmin(s.confirmProposal))
	mux.HandleFunc("POST /proposals/decline", s.requireAdmin(s.declineLookup))

	// Folding a viewer-readable read into admin Settings would downgrade a viewer's access (#281).
	mux.HandleFunc("GET /coverage", s.requireLogin(s.coveragePage))

	mux.HandleFunc("GET /scans", s.requireLogin(s.scansPage))
	mux.HandleFunc("POST /scans/trigger", s.requireAdmin(s.triggerScan))
	mux.HandleFunc("POST /scans/stop", s.requireAdmin(s.stopScan))
	mux.HandleFunc("POST /scans/terminate", s.requireAdmin(s.terminateScan))

	mux.HandleFunc("GET /messages", s.requireLogin(s.messagesPage))
	mux.HandleFunc("POST /messages/read", s.requireLogin(s.markMessageRead))
	mux.HandleFunc("POST /messages/read-all", s.requireLogin(s.markAllMessagesRead))
	mux.HandleFunc("POST /messages/unread", s.requireLogin(s.markMessageUnread))

	mux.HandleFunc("GET /inbox", s.requireLogin(s.inboxPage))

	mux.HandleFunc("GET /verge-core", s.requireLogin(s.vergeCorePage))
	mux.HandleFunc("POST /verge-core/frequency", s.requireAdmin(s.editVergeCoreFrequency))

	mux.HandleFunc("GET /sources", s.requireLogin(s.sourcesModal))
	mux.HandleFunc("POST /sources/toggle", s.requireAdmin(s.toggleSource))
	mux.HandleFunc("POST /settings/sources", s.requireAdmin(s.settingsSources))

	mux.HandleFunc("GET /profile", s.requireLogin(s.profilePage))
	mux.HandleFunc("POST /profile/password", s.requireLogin(s.changePassword))
	mux.HandleFunc("POST /profile/tokens", s.requireLogin(s.createPersonalToken))
	mux.HandleFunc("POST /profile/tokens/revoke", s.requireLogin(s.revokePersonalToken))
	mux.HandleFunc("POST /profile/session/revoke", s.requireLogin(s.revokeSession))
	mux.HandleFunc("POST /profile/sessions/revoke", s.requireLogin(s.revokeOneSession))
	mux.HandleFunc("POST /profile/sessions/revoke-others", s.requireLogin(s.signOutOtherSessions))
	mux.HandleFunc("GET /profile/sso/{slug}/link", s.requireLogin(s.ssoLinkStart))
	mux.HandleFunc("GET /profile/sso/{slug}/link/callback", s.requireLogin(s.ssoLinkCallback))
	mux.HandleFunc("POST /profile/sso/unlink", s.requireLogin(s.ssoUnlink))

	mux.HandleFunc("GET /account", s.requireLogin(s.accountPage))
	mux.HandleFunc("POST /accounts", s.requireAdmin(s.createAccount))
	mux.HandleFunc("GET /account/totp/enroll", s.requireLogin(s.totpEnrollForm))
	mux.HandleFunc("POST /account/totp/enable", s.requireLogin(s.totpEnable))
	mux.HandleFunc("POST /account/totp/confirm", s.requireLogin(s.totpConfirm))

	mux.HandleFunc("GET /settings", s.requireSettingsAdmin(s.settingsPage))
	mux.HandleFunc("POST /settings/accounts", s.requireAdmin(s.inviteAccount))
	mux.HandleFunc("POST /settings/accounts/role", s.requireAdmin(s.setAccountRole))
	mux.HandleFunc("POST /settings/accounts/reenroll", s.requireAdmin(s.reenrollAccount))
	mux.HandleFunc("POST /settings/accounts/remove", s.requireAdmin(s.removeAccount))
	mux.HandleFunc("POST /settings/sessions/revoke", s.requireAdmin(s.revokeSessionAdmin))
	mux.HandleFunc("POST /settings/sessions/revoke-account", s.requireAdmin(s.revokeAccountSessions))
	mux.HandleFunc("POST /settings/channels", s.requireAdmin(s.createChannel))
	mux.HandleFunc("POST /settings/channels/update", s.requireAdmin(s.updateChannel))
	mux.HandleFunc("POST /settings/channels/delete", s.requireAdmin(s.deleteChannel))
	mux.HandleFunc("POST /settings/channels/test", s.requireAdmin(s.testChannel))
	mux.HandleFunc("POST /settings/retention", s.requireAdmin(s.updateRetention))
	mux.HandleFunc("POST /settings/address-cap", s.requireAdmin(s.updateAddressCap))
	mux.HandleFunc("POST /settings/updates/check", s.requireAdmin(s.updateCheckToggle))
	mux.HandleFunc("POST /settings/backup", s.requireAdmin(s.backupDownload))
	mux.HandleFunc("POST /settings/restore/preflight", s.requireAdmin(s.restorePreflight))
	mux.HandleFunc("POST /settings/restore", s.requireAdmin(s.restoreApply))
	mux.HandleFunc("POST /settings/api", s.requireAdmin(s.apiToggle))

	mux.HandleFunc("POST /settings/sso", s.requireAdmin(s.createSSOProvider))
	mux.HandleFunc("POST /settings/sso/update", s.requireAdmin(s.updateSSOProvider))
	mux.HandleFunc("POST /settings/sso/secret", s.requireAdmin(s.setSSOProviderSecret))
	mux.HandleFunc("POST /settings/sso/delete", s.requireAdmin(s.deleteSSOProvider))
	mux.HandleFunc("POST /settings/sso/identity/remove", s.requireAdmin(s.removeSSOBinding))

	if integrationsEnabled {
		// With the surface off no user-facing route can write integration_state at all (#388).
		mux.HandleFunc("POST /settings/integrations/install", s.requireAdmin(s.installIntegration))
		mux.HandleFunc("POST /settings/integrations/remove", s.requireAdmin(s.removeIntegration))
		mux.HandleFunc("POST /settings/integrations/disconnect", s.requireAdmin(s.removeIntegration))
		mux.HandleFunc("POST /settings/integrations/test", s.requireAdmin(s.testIntegration))
		mux.HandleFunc("POST /settings/integrations/channel", s.requireAdmin(s.bindIntegrationChannel))
	}

	if s.devMode {
		mux.HandleFunc("GET /dev/403", s.forbidden)
		mux.HandleFunc("GET /dev/panic", s.devPanic)
		mux.HandleFunc("GET /dev/session/{role}", s.devSessionMint)
		mux.HandleFunc("GET /dev/profile/session", s.devProfileSessionPrepare)
		mux.HandleFunc("GET /dev/seed/empty", s.devSetupSeedEmpty)
		mux.HandleFunc("GET /dev/seed/empty-authed", s.devCoverageSeedEmpty)
	}

	s.mountAPIv1(mux)

	// Set after the last route, so the submitting-URL guard sees every route (ADR-0171 §1).
	s.routes = mux

	return s.recoverPanics(mux)
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	hb, err := s.healthzStore.RecordHeartbeat(r.Context())
	if err != nil {
		log.Printf("web: healthz: record heartbeat: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status    string    `json:"status"`
		CheckedAt time.Time `json:"checked_at"`
	}{Status: "ok", CheckedAt: hb.CheckedAt.Time})
}
