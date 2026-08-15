package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/proposer"
	"github.com/winniel123/verge-asm/internal/seed"
)

// store is the slice of the database the web handlers use. Narrowing it to an
// interface lets tests supply an in-memory fake instead of a live Postgres,
// the same seam the scaffold's healthz handler introduced.
type store interface {
	RecordHeartbeat(ctx context.Context) (db.Heartbeat, error)
	CountAccounts(ctx context.Context) (int64, error)
	CreateAccount(ctx context.Context, arg db.CreateAccountParams) (db.Account, error)
	GetAccountByUsername(ctx context.Context, username string) (db.Account, error)
	GetAccountByID(ctx context.Context, id int64) (db.Account, error)
	SetTOTPSecret(ctx context.Context, arg db.SetTOTPSecretParams) error
	ConfirmTOTP(ctx context.Context, id int64) error
	CreateNameSeed(ctx context.Context, arg db.CreateNameSeedParams) (db.Seed, error)
	CreateAddressSeed(ctx context.Context, arg db.CreateAddressSeedParams) (db.Seed, error)
	ListSeeds(ctx context.Context) ([]db.ListSeedsRow, error)
	SetCustodyExtension(ctx context.Context, arg db.SetCustodyExtensionParams) error
	CreateNameExclusion(ctx context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error)
	CreateAddressExclusion(ctx context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error)
	ListExclusions(ctx context.Context) ([]db.ListExclusionsRow, error)
	DeleteExclusion(ctx context.Context, id int64) error
	ListSourceStates(ctx context.Context) ([]db.ListSourceStatesRow, error)
	UpsertSourceState(ctx context.Context, arg db.UpsertSourceStateParams) (db.SourceState, error)
	CreateVantage(ctx context.Context, arg db.CreateVantageParams) (db.Vantage, error)
	ListVantages(ctx context.Context) ([]db.ListVantagesRow, error)
	GetScanByKind(ctx context.Context, kind string) (db.Scan, error)
	ListAccounts(ctx context.Context) ([]db.ListAccountsRow, error)
	CountAdmins(ctx context.Context) (int64, error)
	UpdateAccountRole(ctx context.Context, arg db.UpdateAccountRoleParams) error
	CreateChannel(ctx context.Context, arg db.CreateChannelParams) (int64, error)
	ListChannels(ctx context.Context) ([]db.ListChannelsRow, error)
	UpdateChannel(ctx context.Context, arg db.UpdateChannelParams) error
	SetChannelSecret(ctx context.Context, arg db.SetChannelSecretParams) error
	DeleteChannel(ctx context.Context, id int64) error
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	UpdateRetentionSettings(ctx context.Context, arg db.UpdateRetentionSettingsParams) error
	// TightestEnabledScanCadenceSeconds is the tightest bound in force, which the
	// observation dial floors at (#208, ADR-0094) — symmetric to the Dispatch
	// floor's SlowestEnabledScanCadenceSeconds.
	TightestEnabledScanCadenceSeconds(ctx context.Context) (int64, error)
	ListCurrentNameSubjects(ctx context.Context, search string) ([]db.ListCurrentNameSubjectsRow, error)
	GetNameSubject(ctx context.Context, subjectKey string) (db.GetNameSubjectRow, error)
	GetNameCitation(ctx context.Context, subjectKey string) (db.GetNameCitationRow, error)
	FindCoveringNameSeed(ctx context.Context, name string) (db.FindCoveringNameSeedRow, error)
	ListCurrentServiceSubjects(ctx context.Context, search string) ([]db.ListCurrentServiceSubjectsRow, error)
	GetServiceSubject(ctx context.Context, subjectKey string) (db.GetServiceSubjectRow, error)
	// Endpoint subjects (#198): the (Name, Service) pair the http-exchange leaf's
	// http-identity facet is held on. Rendered on the Subjects page additively next
	// to Name and Service, with its own drill-down.
	ListCurrentEndpointSubjects(ctx context.Context, search string) ([]db.ListCurrentEndpointSubjectsRow, error)
	GetEndpointSubject(ctx context.Context, subjectKey string) (db.GetEndpointSubjectRow, error)
	FindNameCitingAddress(ctx context.Context, address string) (db.FindNameCitingAddressRow, error)
	FindCoveringAddressSeed(ctx context.Context, address netip.Addr) (db.FindCoveringAddressSeedRow, error)
	ListSpansForSubject(ctx context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error)
	// Exposure landing view (#196): the two most recent reachability spans per
	// (Service, vantage), joined to the prober endpoint. The class is re-verified
	// per render from the presented address, so this read carries the host rather
	// than the static vantage class.
	ListReachabilitySpansForExposure(ctx context.Context) ([]db.ListReachabilitySpansForExposureRow, error)
	CreateZoneFile(ctx context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error)
	ListZoneFileStatus(ctx context.Context) ([]db.ListZoneFileStatusRow, error)
	GetZoneCadenceSeconds(ctx context.Context) (int64, error)
	SetZoneCadenceSeconds(ctx context.Context, cadenceSeconds int64) error
	// The cold Scan opt-in (#200): the full-range tier is enabled per-Seed, not
	// globally. Opting a scope in or out reconciles the Scan's enabled flag, which
	// is what puts it on — or takes it off — the dispatcher's cadence.
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
	ListVergeCoreFrequencyEditsWithAuthor(ctx context.Context) ([]db.ListVergeCoreFrequencyEditsWithAuthorRow, error)
	UpsertVergeCoreFrequencyEdit(ctx context.Context, arg db.UpsertVergeCoreFrequencyEditParams) error
	DeleteVergeCoreFrequencyEdit(ctx context.Context, port int32) error
	// Signals reads (#202): the Derived corpus the Signal engine folds into its
	// per-Name snapshot — resolution per Vantage class, the dns-record CNAME/NS
	// records, and the operator's zone declarations.
	ListNameResolutionsByClass(ctx context.Context) ([]db.ListNameResolutionsByClassRow, error)
	ListNameDNSRecords(ctx context.Context) ([]db.ListNameDNSRecordsRow, error)
	ListZoneDeclarations(ctx context.Context) ([]db.ListZoneDeclarationsRow, error)
	// Annotation management (#204): an operator dial keyed on one
	// (subject, signal-name) pair, carrying the reason and the declared instant —
	// no status, no expiry, no author. Declaring and withdrawing mint no Message.
	CreateAnnotation(ctx context.Context, arg db.CreateAnnotationParams) (db.Annotation, error)
	ListAnnotations(ctx context.Context) ([]db.Annotation, error)
	DeleteAnnotation(ctx context.Context, id int64) error
	// The global message panel (#205): the Message store is unconditional — every
	// message is written and rendered, and the nav element carries the unread
	// count on every screen. There is no delete and no content update; a message
	// is computed once at the cause and read back verbatim.
	InsertMessage(ctx context.Context, arg db.InsertMessageParams) (db.Message, error)
	ListMessages(ctx context.Context) ([]db.Message, error)
	CountUnreadMessages(ctx context.Context) (int64, error)
	MarkMessageRead(ctx context.Context, arg db.MarkMessageReadParams) error
	MarkAllMessagesRead(ctx context.Context, readAt pgtype.Timestamptz) error
	// PreviewExclusionWithdrawal counts the subjects a candidate exclusion would
	// withdraw and the timelines they hold — the honestly-computable narrowing
	// receipt (#205 AC8, ADR-0074). It reads only ground nothing else cites, so a
	// subject a current resolution still holds is not counted (its Gap carries it).
	PreviewExclusionWithdrawal(ctx context.Context, arg db.PreviewExclusionWithdrawalParams) (db.PreviewExclusionWithdrawalRow, error)
}

// server holds everything the handlers need: the database, the session signing
// key (read from the web-only volume, never Postgres), the active single-use
// setup token (empty once an admin exists), and an injectable clock.
type server struct {
	store      store
	key        []byte
	setupToken string
	now        func() time.Time
	sessionTTL time.Duration
	pendingTTL time.Duration
	// seedAddressCap is the ceiling on addresses an address-scope Seed may
	// cover. It defaults to seed.DefaultAddressCap; the Settings screen (#206)
	// will make it operator-configurable.
	seedAddressCap int

	// proposer runs the enabled keyless registry proposer paths for one operator
	// lookup (ADR-0012). It defaults to the shipped registry over a real HTTP
	// client; tests inject a fake so no lookup touches the network.
	proposer proposerRunner

	// secureCookies forces the Secure attribute on auth cookies even when the
	// request did not itself arrive over TLS — set it when web is fronted by a
	// TLS-terminating proxy (VERGE_SECURE_COOKIES).
	secureCookies bool
	// setupMu serialises the first-boot setup so a concurrent pair of valid
	// POST /setup requests cannot both pass the no-accounts check and each
	// create an admin, which would break the token's single-use guarantee.
	setupMu sync.Mutex
}

func newServer(s store, key []byte, setupToken string, now func() time.Time) *server {
	return &server{
		store:          s,
		key:            key,
		setupToken:     setupToken,
		now:            now,
		sessionTTL:     12 * time.Hour,
		pendingTTL:     5 * time.Minute,
		seedAddressCap: seed.DefaultAddressCap,
		proposer:       proposer.DefaultRegistry(&http.Client{Timeout: 30 * time.Second}),
	}
}

// handler wires every route. A permission check runs on every mutating
// endpoint from this commit forward (v1 spec §4.3): the only mutation that
// exists yet, POST /accounts, is gated behind requireAdmin.
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

	mux.HandleFunc("GET /seeds", s.requireLogin(s.seedsPage))
	mux.HandleFunc("POST /seeds", s.requireAdmin(s.declareSeed))
	mux.HandleFunc("POST /seeds/custody", s.requireAdmin(s.setCustody))
	mux.HandleFunc("POST /seeds/zone", s.requireAdmin(s.uploadZoneFile))
	mux.HandleFunc("POST /seeds/zone/interval", s.requireAdmin(s.setZoneInterval))
	mux.HandleFunc("POST /seeds/cold", s.requireAdmin(s.setColdScope))
	mux.HandleFunc("POST /exclusions", s.requireAdmin(s.declareExclusion))
	mux.HandleFunc("POST /exclusions/preview", s.requireAdmin(s.previewExclusion))
	mux.HandleFunc("POST /exclusions/delete", s.requireAdmin(s.unexclude))
	mux.HandleFunc("POST /probers", s.requireAdmin(s.provisionProber))

	// The Exposure landing view (v1 spec §6.2): the exposure board, a read-only
	// projection over the reachability corpus. A dedicated nav destination; the
	// board itself is a census and never an alert source (ADR-0029).
	mux.HandleFunc("GET /exposure", s.requireLogin(s.exposurePage))

	mux.HandleFunc("GET /subjects", s.requireLogin(s.subjectsPage))
	// The Service and Endpoint drill-downs read their key from a query parameter
	// because those keys carry `/` and `@`; their literal paths win over the {key}
	// wildcard.
	mux.HandleFunc("GET /subjects/service", s.requireLogin(s.servicePage))
	mux.HandleFunc("GET /subjects/endpoint", s.requireLogin(s.endpointPage))
	mux.HandleFunc("GET /subjects/{key}", s.requireLogin(s.subjectPage))

	mux.HandleFunc("GET /signals", s.requireLogin(s.signalsPage))
	mux.HandleFunc("POST /annotations", s.requireAdmin(s.declareAnnotation))
	mux.HandleFunc("POST /annotations/withdraw", s.requireAdmin(s.withdrawAnnotation))

	// Registry proposer lookups and the confirm/decline of the Proposals they
	// yield are admin acts (v1 spec §4.3): confirming opens the probing gate on
	// address space, declining is a boundary claim. A viewer reads the pending
	// list on /seeds but mutates nothing.
	mux.HandleFunc("POST /proposals", s.requireAdmin(s.runLookup))
	mux.HandleFunc("POST /proposals/confirm", s.requireAdmin(s.confirmProposal))
	mux.HandleFunc("POST /proposals/decline", s.requireAdmin(s.declineLookup))

	mux.HandleFunc("GET /coverage", s.requireLogin(s.coveragePage))

	// The global message panel (#205, v1 spec §6.7): a viewer reads the unbounded
	// list and its unread count on every screen; marking read is a per-account
	// read-state change, so a viewer may do it. The store is unconditional and has
	// no admin surface — there is nothing here to gate behind requireAdmin.
	mux.HandleFunc("GET /messages", s.requireLogin(s.messagesPage))
	mux.HandleFunc("POST /messages/read", s.requireLogin(s.markMessageRead))
	mux.HandleFunc("POST /messages/read-all", s.requireLogin(s.markAllMessagesRead))

	// verge-core: a viewer reads the composed set; editing the frequency half is
	// an admin act (v1 spec §3.5, §4.3). The sensitive half is authored by the
	// release and has no mutating endpoint at all.
	mux.HandleFunc("GET /verge-core", s.requireLogin(s.vergeCorePage))
	mux.HandleFunc("POST /verge-core/frequency", s.requireAdmin(s.editVergeCoreFrequency))

	mux.HandleFunc("GET /sources", s.requireLogin(s.sourcesModal))
	mux.HandleFunc("POST /sources/toggle", s.requireAdmin(s.toggleSource))

	mux.HandleFunc("POST /accounts", s.requireAdmin(s.createAccount))
	mux.HandleFunc("POST /account/totp/enable", s.requireLogin(s.totpEnable))
	mux.HandleFunc("POST /account/totp/confirm", s.requireLogin(s.totpConfirm))

	// The Settings destination and every mutation it hosts are admin acts
	// (v1 spec §4.3, §6.1): viewing the operator's dials and moving them are
	// both gated behind requireAdmin.
	mux.HandleFunc("GET /settings", s.requireAdmin(s.settingsPage))
	mux.HandleFunc("POST /settings/accounts", s.requireAdmin(s.inviteAccount))
	mux.HandleFunc("POST /settings/accounts/role", s.requireAdmin(s.setAccountRole))
	mux.HandleFunc("POST /settings/channels", s.requireAdmin(s.createChannel))
	mux.HandleFunc("POST /settings/channels/update", s.requireAdmin(s.updateChannel))
	mux.HandleFunc("POST /settings/channels/delete", s.requireAdmin(s.deleteChannel))
	mux.HandleFunc("POST /settings/retention", s.requireAdmin(s.updateRetention))
	return mux
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
