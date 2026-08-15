package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
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
	ListCurrentNameSubjects(ctx context.Context, search string) ([]db.ListCurrentNameSubjectsRow, error)
	GetNameSubject(ctx context.Context, subjectKey string) (db.GetNameSubjectRow, error)
	GetNameCitation(ctx context.Context, subjectKey string) (db.GetNameCitationRow, error)
	FindCoveringNameSeed(ctx context.Context, name string) (db.FindCoveringNameSeedRow, error)
	CreateZoneFile(ctx context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error)
	ListZoneFileStatus(ctx context.Context) ([]db.ListZoneFileStatusRow, error)
	GetZoneCadenceSeconds(ctx context.Context) (int64, error)
	SetZoneCadenceSeconds(ctx context.Context, cadenceSeconds int64) error
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
	mux.HandleFunc("POST /exclusions", s.requireAdmin(s.declareExclusion))
	mux.HandleFunc("POST /exclusions/delete", s.requireAdmin(s.unexclude))
	mux.HandleFunc("POST /probers", s.requireAdmin(s.provisionProber))

	mux.HandleFunc("GET /subjects", s.requireLogin(s.subjectsPage))
	mux.HandleFunc("GET /subjects/{key}", s.requireLogin(s.subjectPage))

	mux.HandleFunc("GET /coverage", s.requireLogin(s.coveragePage))
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
