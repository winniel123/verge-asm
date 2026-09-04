package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/netip"
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

type store interface {
	RecordHeartbeat(ctx context.Context) (db.Heartbeat, error)
	CountAccounts(ctx context.Context) (int64, error)
	CreateAccount(ctx context.Context, arg db.CreateAccountParams) (db.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (db.Account, error)
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	SetTOTPSecret(ctx context.Context, arg db.SetTOTPSecretParams) error
	ConfirmTOTP(ctx context.Context, id int64) error
	SetTOTPLastStep(ctx context.Context, arg db.SetTOTPLastStepParams) (int64, error)
	UpdatePassword(ctx context.Context, arg db.UpdatePasswordParams) error
	CreatePersonalToken(ctx context.Context, arg db.CreatePersonalTokenParams) (db.PersonalToken, error)
	ListPersonalTokens(ctx context.Context, accountID int64) ([]db.ListPersonalTokensRow, error)
	DeletePersonalToken(ctx context.Context, arg db.DeletePersonalTokenParams) error
	CreatePasswordReset(ctx context.Context, arg db.CreatePasswordResetParams) (db.PasswordReset, error)
	GetPasswordResetByHash(ctx context.Context, tokenHash string) (db.PasswordReset, error)
	ConsumePasswordReset(ctx context.Context, arg db.ConsumePasswordResetParams) error
	CreateRecoveryCode(ctx context.Context, arg db.CreateRecoveryCodeParams) error
	DeleteRecoveryCodesForAccount(ctx context.Context, accountID int64) error
	ListUnusedRecoveryCodeHashes(ctx context.Context, accountID int64) ([]db.ListUnusedRecoveryCodeHashesRow, error)
	ConsumeRecoveryCode(ctx context.Context, arg db.ConsumeRecoveryCodeParams) error
	GetInviteByTokenHash(ctx context.Context, tokenHash string) (db.Invite, error)
	ConsumeInvite(ctx context.Context, arg db.ConsumeInviteParams) error
	CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error)
	GetSessionByTokenHash(ctx context.Context, arg db.GetSessionByTokenHashParams) (db.Session, error)
	TouchSession(ctx context.Context, arg db.TouchSessionParams) error
	RevokeSession(ctx context.Context, arg db.RevokeSessionParams) error
	ListSessionsForAccount(ctx context.Context, arg db.ListSessionsForAccountParams) ([]db.ListSessionsForAccountRow, error)
	RevokeOtherSessionsForAccount(ctx context.Context, arg db.RevokeOtherSessionsForAccountParams) error
	RevokeAllSessionsForAccount(ctx context.Context, arg db.RevokeAllSessionsForAccountParams) error
	ListAllActiveSessions(ctx context.Context, expiresAt pgtype.Timestamptz) ([]db.ListAllActiveSessionsRow, error)
	RevokeSessionByIDForAdmin(ctx context.Context, arg db.RevokeSessionByIDForAdminParams) error
	CreateInvite(ctx context.Context, arg db.CreateInviteParams) (db.Invite, error)
	ResetAccountTOTP(ctx context.Context, id int64) error
	DeleteAccount(ctx context.Context, id int64) error
	CreateNameSeed(ctx context.Context, arg db.CreateNameSeedParams) (db.Seed, error)
	CreateAddressSeed(ctx context.Context, arg db.CreateAddressSeedParams) (db.Seed, error)
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	WithdrawSeed(ctx context.Context, arg db.WithdrawSeedParams) (db.WithdrawSeedRow, error)
	ListSeedWithdrawalCandidates(ctx context.Context, cidrs []string) ([]db.ListSeedWithdrawalCandidatesRow, error)
	ListNameSeedWithdrawalCandidates(ctx context.Context, domains []string) ([]db.ListNameSeedWithdrawalCandidatesRow, error)
	ListAdmittedNamesOutsideSeed(ctx context.Context, seedID int64) ([]string, error)
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) error
	CreateNameExclusion(ctx context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error)
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	ListExclusions(ctx context.Context) ([]db.ListExclusionsRow, error)
	DeleteExclusion(ctx context.Context, id int64) error
	ListSourceStates(ctx context.Context) ([]db.SourceState, error)
	UpsertSourceState(ctx context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error)
	CTReliabilityWindow(ctx context.Context, arg db.CTReliabilityWindowParams) (db.CTReliabilityWindowRow, error)
	CTLastBatchAdmitCount(ctx context.Context) (int64, error)
	CTTailLastBatch(ctx context.Context) (db.CTTailLastBatchRow, error)
	CountCertificateMaterial(ctx context.Context) (int64, error)
	ListIntegrationStates(ctx context.Context) ([]db.IntegrationState, error)
	UpsertIntegrationState(ctx context.Context, arg db.UpsertIntegrationStateParams) (db.IntegrationState, error)
	DeleteIntegrationState(ctx context.Context, slug string) error
	GetIntegrationChannel(ctx context.Context, slug string) (pgtype.Int8, error)
	SetIntegrationChannel(ctx context.Context, arg db.SetIntegrationChannelParams) error
	GetChannelForDelivery(ctx context.Context, id int64) (db.GetChannelForDeliveryRow, error)
	CreateVantage(ctx context.Context, arg db.CreateVantageParams) (db.Vantage, error)
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListAddressExclusionCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListExtendedZoneDomains(ctx context.Context) ([]pgtype.Text, error)
	NameCitedAddresses(ctx context.Context, arg db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error)
	ListEdgeFanoutMeasurements(ctx context.Context) ([]db.ListEdgeFanoutMeasurementsRow, error)
	ListEdgeFanoutMeasurementsOver(ctx context.Context, addresses []string) ([]db.ListEdgeFanoutMeasurementsOverRow, error)
	ListCertificateMaterialDER(ctx context.Context, fingerprints []string) ([]db.ListCertificateMaterialDERRow, error)
	ScanHasCompletedBatch(ctx context.Context, kind string) (bool, error)
	ListUnavailableVantages(ctx context.Context) ([]db.ListUnavailableVantagesRow, error)
	GetScanByKind(ctx context.Context, kind string) (db.Scan, error)
	ListAccounts(ctx context.Context) ([]db.ListAccountsRow, error)
	CountAdmins(ctx context.Context) (int64, error)
	UpdateAccountRole(ctx context.Context, arg db.UpdateAccountRoleParams) error
	CreateChannel(ctx context.Context, arg db.CreateChannelParams) (int64, error)
	ListChannels(ctx context.Context) ([]db.ListChannelsRow, error)
	UpdateChannel(ctx context.Context, arg db.UpdateChannelParams) error
	SetChannelSecret(ctx context.Context, arg db.SetChannelSecretParams) error
	DeleteChannel(ctx context.Context, id int64) error
	GetInstanceHealth(ctx context.Context) (db.GetInstanceHealthRow, error)
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	UpdateRetentionSettings(ctx context.Context, arg db.UpdateRetentionSettingsParams) error
	GetInstanceConfig(ctx context.Context) (db.GetInstanceConfigRow, error)
	SetUpdateCheckEnabled(ctx context.Context, arg db.SetUpdateCheckEnabledParams) error
	SetLastBackup(ctx context.Context, lastBackupSize pgtype.Int8) error
	SetAPIEnabled(ctx context.Context, arg db.SetAPIEnabledParams) error
	SetSeedAddressCap(ctx context.Context, arg db.SetSeedAddressCapParams) error
	TightestEnabledScanCadenceSeconds(ctx context.Context) (int64, error)
	ListCurrentNameSubjects(ctx context.Context, arg db.ListCurrentNameSubjectsParams) ([]db.ListCurrentNameSubjectsRow, error)
	GetNameSubject(ctx context.Context, arg db.GetNameSubjectParams) (db.GetNameSubjectRow, error)
	GetNameCitation(ctx context.Context, arg db.GetNameCitationParams) (db.GetNameCitationRow, error)
	FindCoveringNameSeed(ctx context.Context, name string) (db.FindCoveringNameSeedRow, error)
	FindNameSeedByID(ctx context.Context, seedID int64) (db.FindNameSeedByIDRow, error)
	ListCurrentServiceSubjects(ctx context.Context, arg db.ListCurrentServiceSubjectsParams) ([]db.ListCurrentServiceSubjectsRow, error)
	GetServiceSubject(ctx context.Context, arg db.GetServiceSubjectParams) (db.GetServiceSubjectRow, error)
	ListCurrentEndpointSubjects(ctx context.Context, arg db.ListCurrentEndpointSubjectsParams) ([]db.ListCurrentEndpointSubjectsRow, error)
	GetEndpointSubject(ctx context.Context, arg db.GetEndpointSubjectParams) (db.GetEndpointSubjectRow, error)
	FindNameCitingAddress(ctx context.Context, arg db.FindNameCitingAddressParams) (db.FindNameCitingAddressRow, error)
	FindCoveringAddressSeed(ctx context.Context, address netip.Addr) (db.FindCoveringAddressSeedRow, error)
	ListSpansForSubject(ctx context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error)
	ListAllOpenSpans(ctx context.Context) ([]db.ListAllOpenSpansRow, error)
	ListRecentDriftEvents(ctx context.Context, arg db.ListRecentDriftEventsParams) ([]db.ListRecentDriftEventsRow, error)
	ListWithdrawalLifespans(ctx context.Context, since pgtype.Timestamptz) ([]db.ListWithdrawalLifespansRow, error)
	ListSubjectFirstAppearances(ctx context.Context, since pgtype.Timestamptz) ([]db.ListSubjectFirstAppearancesRow, error)
	PreviousBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	EarliestBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	ListSpansOpenSince(ctx context.Context, since pgtype.Timestamptz) ([]db.ListSpansOpenSinceRow, error)
	ListServiceReachabilitySpansByClassAt(ctx context.Context, at pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error)
	CreateZoneFile(ctx context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error)
	ListZoneFileStatus(ctx context.Context) ([]db.ListZoneFileStatusRow, error)
	GetZoneCadenceSeconds(ctx context.Context) (int64, error)
	SetZoneCadenceSeconds(ctx context.Context, cadenceSeconds int64) error
	ListColdScopeSeedIds(ctx context.Context) ([]int64, error)
	OptInColdScope(ctx context.Context, arg db.OptInColdScopeParams) error
	OptOutColdScope(ctx context.Context, seedID int64) error
	SyncColdScanEnabled(ctx context.Context) error
	CreateProposerLookup(ctx context.Context, arg db.CreateProposerLookupParams) (db.ProposerLookup, error)
	CreateProposal(ctx context.Context, arg db.CreateProposalParams) (db.Proposal, error)
	ListPendingProposals(ctx context.Context) ([]db.ListPendingProposalsRow, error)
	GetPendingProposal(ctx context.Context, id int64) (db.Proposal, error)
	ConfirmProposal(ctx context.Context, arg db.ConfirmProposalParams) (int64, error)
	DeclineLookup(ctx context.Context, lookupID int64) (int64, error)
	DeclineProposal(ctx context.Context, id int64) (int64, error)
	ListVergeCoreFrequencyEditsWithAuthor(ctx context.Context) ([]db.ListVergeCoreFrequencyEditsWithAuthorRow, error)
	UpsertVergeCoreFrequencyEdit(ctx context.Context, arg db.UpsertVergeCoreFrequencyEditParams) error
	DeleteVergeCoreFrequencyEdit(ctx context.Context, port int32) error
	ListNameResolutionsByClass(ctx context.Context, arg db.ListNameResolutionsByClassParams) ([]db.ListNameResolutionsByClassRow, error)
	ListNameDNSRecords(ctx context.Context, arg db.ListNameDNSRecordsParams) ([]db.ListNameDNSRecordsRow, error)
	ListZoneDeclarations(ctx context.Context) ([]db.ListZoneDeclarationsRow, error)
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListServiceTLSAcceptance(ctx context.Context, arg db.ListServiceTLSAcceptanceParams) ([]db.ListServiceTLSAcceptanceRow, error)
	ListBlanketedReachServices(ctx context.Context) ([]string, error)
	ListEndpointCertificates(ctx context.Context, arg db.ListEndpointCertificatesParams) ([]db.ListEndpointCertificatesRow, error)
	CreateAnnotation(ctx context.Context, arg db.CreateAnnotationParams) (db.Annotation, error)
	ListAnnotations(ctx context.Context) ([]db.Annotation, error)
	DeleteAnnotation(ctx context.Context, id int64) error
	MintSignalInstances(ctx context.Context, arg db.MintSignalInstancesParams) error
	ListSignalInstances(ctx context.Context) ([]db.SignalInstance, error)
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
	ListMessages(ctx context.Context) ([]db.Message, error)
	ListReadMessageIDs(ctx context.Context, accountID int64) ([]int64, error)
	CountUnreadMessages(ctx context.Context, accountID int64) (int64, error)
	ListDeliveryOutcomes(ctx context.Context) ([]db.ListDeliveryOutcomesRow, error)
	MarkMessageRead(ctx context.Context, arg db.MarkMessageReadParams) error
	MarkAllMessagesRead(ctx context.Context, arg db.MarkAllMessagesReadParams) error
	MarkMessageUnread(ctx context.Context, arg db.MarkMessageUnreadParams) error
	PreviewExclusionWithdrawal(ctx context.Context, arg db.PreviewExclusionWithdrawalParams) (db.PreviewExclusionWithdrawalRow, error)
	ListDispatchProgress(ctx context.Context, limit int32) ([]db.ListDispatchProgressRow, error)
	ListJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) ([]db.ListJobsForDispatchRow, error)
	ListActiveDispatchProgress(ctx context.Context) ([]db.ListActiveDispatchProgressRow, error)
	ListConcludedDispatchProgress(ctx context.Context, limit int32) ([]db.ListConcludedDispatchProgressRow, error)
	GetTranscriptByJob(ctx context.Context, queueJobID int64) (db.Transcript, error)
	CancelReadyJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) (int64, error)
	CancelActiveJobsForDispatch(ctx context.Context, dispatchID pgtype.Int8) (int64, error)
	SetDispatchStatus(ctx context.Context, arg db.SetDispatchStatusParams) error
	ListScans(ctx context.Context) ([]db.Scan, error)

	InsertReportSchedule(ctx context.Context, arg db.InsertReportScheduleParams) (db.ReportSchedule, error)
	ListReportSchedules(ctx context.Context) ([]db.ReportSchedule, error)
	GetReportSchedule(ctx context.Context, id int64) (db.ReportSchedule, error)
	UpdateReportSchedule(ctx context.Context, arg db.UpdateReportScheduleParams) (db.ReportSchedule, error)
	DeleteReportSchedule(ctx context.Context, id int64) error

	InsertReportDelivery(ctx context.Context, arg db.InsertReportDeliveryParams) (db.ReportDelivery, error)
	NextReportDeliveryNo(ctx context.Context, scheduleID int64) (int32, error)
	GetLatestReportDelivery(ctx context.Context, scheduleID int64) (db.ReportDelivery, error)
	ListReportDeliveries(ctx context.Context, scheduleID int64) ([]db.ReportDelivery, error)

	InsertSSOProvider(ctx context.Context, arg db.InsertSSOProviderParams) (int64, error)
	ListSSOProviders(ctx context.Context) ([]db.ListSSOProvidersRow, error)
	ListEnabledSSOProviders(ctx context.Context) ([]db.ListEnabledSSOProvidersRow, error)
	GetSSOProvider(ctx context.Context, id int64) (db.GetSSOProviderRow, error)

	// A secret is read only where its act is performed, so no listing read may select it (ADR-0053).

	GetSSOProviderForAuth(ctx context.Context, slug string) (db.GetSSOProviderForAuthRow, error)
	UpdateSSOProvider(ctx context.Context, arg db.UpdateSSOProviderParams) (int64, error)
	SetSSOProviderSecret(ctx context.Context, arg db.SetSSOProviderSecretParams) error
	DeleteSSOProvider(ctx context.Context, id int64) error

	InsertSSOIdentity(ctx context.Context, arg db.InsertSSOIdentityParams) error
	GetAccountBySSOIdentity(ctx context.Context, arg db.GetAccountBySSOIdentityParams) (db.Account, error)
	GetSSOIdentityBySub(ctx context.Context, arg db.GetSSOIdentityBySubParams) (db.GetSSOIdentityBySubRow, error)
	ListSSOIdentitiesForAccount(ctx context.Context, accountID int64) ([]db.ListSSOIdentitiesForAccountRow, error)
	DeleteSSOIdentityForAccount(ctx context.Context, arg db.DeleteSSOIdentityForAccountParams) (int64, error)
	ListSSOBindings(ctx context.Context) ([]db.ListSSOBindingsRow, error)
	DeleteSSOIdentity(ctx context.Context, id int64) error
}

type server struct {
	store store
	key   []byte

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

	// A forwarded-for header is caller-supplied, so it keys the rate limiter and never authorization.

	trustedProxies trustedProxies

	// The Host header is attacker-controlled, so the OIDC redirect_uri never derives from it (#293).

	externalURL string

	// Two concurrent valid setups would each pass the no-accounts check and spend the token twice.

	setupMu sync.Mutex

	flash *flashStore

	formFlash *formFlashStore

	loginLimiter *loginLimiter

	progress progressEvents

	// sqlc generates no goose_db_version read, so the raw pool serves what internal/db cannot (#391).

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
	// A nil key fails closed on the encrypt path rather than silently storing cleartext (#337).
	totpKey, _ := auth.DeriveTOTPKey(key)
	return &server{
		store:         s,
		key:           key,
		totpKey:       totpKey,
		setupToken:    setupToken,
		now:           now,
		startedAt:     now(),
		sessionTTL:    12 * time.Hour,
		pendingTTL:    5 * time.Minute,
		resetTTL:      30 * time.Minute,
		proposer:      proposer.DefaultRegistry(&http.Client{Timeout: 30 * time.Second}),
		sso:           newOIDCFlow(&http.Client{Timeout: 30 * time.Second}),
		channelSender: newHTTPChannelSender(now),
		loginLimiter:  newLoginLimiter(now),
		flash:         newFlashStore(),
		formFlash:     newFormFlashStore(),
		restoreStage:  make(map[int64]*restoreStaging),
	}
}

func (s *server) obsAsOf() pgtype.Timestamptz {
	// An evidential row is discardable at any age, so a derivation reads the live tier (ADR-0041).
	return pgtype.Timestamptz{Time: s.now().UTC(), Valid: true}
}

func (s *server) addressCap(ctx context.Context) int {
	// The cap is read at declaration only, so lowering it invalidates no declared scope (ADR-0127).
	cfg, err := s.store.GetInstanceConfig(ctx)
	if err != nil || cfg.SeedAddressCap <= 0 {
		return seed.DefaultAddressCap
	}
	return int(cfg.SeedAddressCap)
}

func (s *server) redirectTo(target string, code int) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, _ db.Account) {
		// A moved route keeps its login gate, so an unauthenticated hit still lands on /login (#286).
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
	// Raw output can carry secrets the redacted log cannot (raw-job-output §5.2), so it is admin-only.
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

	// Set after the last route, so the submitting-URL guard sees every path this server serves.
	s.routes = mux

	return s.recoverPanics(mux)
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	hb, err := s.store.RecordHeartbeat(r.Context())
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
