package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
	"github.com/winniel123/verge-asm/internal/retention"
)

// fakeStore is an in-memory store used across the web handler tests, standing
// in for a live Postgres.
type fakeStore struct {
	hb    db.Heartbeat
	hbErr error

	accounts map[int64]db.Account
	byName   map[string]int64
	nextID   int64

	seeds      []db.Seed
	seedNextID int64

	exclusions []db.Exclusion
	exclNextID int64

	annotations []db.Annotation
	annoNextID  int64

	sourceStates map[string]db.SourceState

	// integrationStates mirrors the integration_state table (#308): the operator's
	// per-integration install state, keyed by slug. Absence is the available (not
	// installed) state, so a disconnect deletes the row.
	integrationStates map[string]db.IntegrationState

	// personalTokens mirrors the personal_token table (#304): an account's own API
	// tokens. Only the hash is stored, the (account_id, name) pair is unique, and a
	// revoke is a hard delete scoped to the owner.
	personalTokens []db.PersonalToken
	tokenNextID    int64

	// SignIn delta (#314): the pre-auth token stores behind forgot/reset, TOTP
	// recovery codes, and invite acceptance. Each keeps only a hash; expiry and
	// single-use are checked against the injected clock, so a fixed-clock test can
	// seed a live or a stale grant deliberately.
	passwordResets []db.PasswordReset
	resetNextID    int64
	recoveryCodes  []db.RecoveryCode
	recoveryNextID int64
	invites        []db.Invite
	inviteNextID   int64

	// admitted stands in for the admitted_name rows behind a CT-admitted Name's
	// Citation (ADR-0107); the citation test seeds it directly.
	admitted []db.AdmittedName

	vantages      []db.Vantage
	vantageNextID int64

	// reachSpans stands in for the reachability-span read behind the Exposure
	// landing view (#196); the exposure test seeds it directly.
	reachSpans []db.ListReachabilitySpansForExposureRow

	channels   []fakeChannel
	chanNextID int64
	retention  db.GetRetentionSettingsRow

	// scans mirrors the scan table. newFakeStore seeds the dns Scan the migration
	// ships (enabled, daily) so the aperture statement has a cadence to read.
	// The Subjects reads (#189) also join the observation/batch corpus.
	observations []db.Observation
	batches      []db.Batch
	scans        []db.Scan
	// listScansErr forces ListScans to fail, so a test can assert the #252 trigger
	// panel degrades to absent rather than 500ing the read-only monitor.
	listScansErr error
	obsNextID    int64
	batchNextID  int64
	scanNextID   int64

	zoneFiles    []fakeZoneFile
	zoneNextID   int64
	zoneCadence  int64
	lookups      []db.ProposerLookup
	lookupNextID int64
	proposals    []db.Proposal
	proposalNext int64

	// freqEdits mirrors the verge_core_frequency_edit table, keyed by port so an
	// upsert replaces the row exactly as the unique index enforces.
	freqEdits map[int32]fakeFreqEdit

	// coldScopes mirrors the cold_scan_scope table: the set of Seed ids opted into
	// the full-range tier, keyed by seed id so an opt-in is idempotent.
	coldScopes map[int64]bool

	// messages mirrors the message table (#205): written once, never updated in
	// content, read back newest-first. previewResult is the fixed narrowing-receipt
	// count a test wants PreviewExclusionWithdrawal to return.
	messages         []db.Message
	deliveryOutcomes []db.ListDeliveryOutcomesRow
	// messageRead mirrors the message_read join table (#327): per-account read-state,
	// keyed account_id -> set of read message ids. Read-state is a per-account fact,
	// so one account marking read never touches another's.
	messageRead   map[int64]map[int64]bool
	msgNextID     int64
	previewResult db.PreviewExclusionWithdrawalRow

	// dispatchProgress and jobsByDispatch stand in for the queue reads behind the
	// Scans monitor (#245); the scans test seeds them directly.
	dispatchProgress []db.ListDispatchProgressRow
	jobsByDispatch   map[int64][]db.ListJobsForDispatchRow

	// reportSchedules mirrors the report_schedule table (#290): the recurring reports
	// the Reports wizard declares, filed once and listed newest-first. No content
	// update and no delete exists, matching the store.
	reportSchedules []db.ReportSchedule
	rsNextID        int64

	// ssoProviders mirrors the sso_provider table (#293): OIDC providers, secret
	// included, so tests can assert the secret is stored but never surfaced through the
	// list/get render paths (only GetSSOProviderForAuth returns it).
	ssoProviders []fakeSSOProvider
	ssoNextID    int64

	// ssoIdentities mirrors the sso_identity table (#319, ADR-0113): the verified
	// (provider, sub) → account bindings authentication keys on.
	ssoIdentities  []fakeSSOIdentity
	ssoIdentNextID int64
}

// fakeSSOProvider mirrors an sso_provider row, client secret included.
type fakeSSOProvider struct {
	id        int64
	slug      string
	name      string
	issuer    string
	clientID  string
	secret    string
	hasSecret bool
	enabled   bool
	createdBy int64
	createdAt time.Time
}

// fakeSSOIdentity mirrors an sso_identity row: a verified subject bound to an account
// through a provider (#319, ADR-0113).
type fakeSSOIdentity struct {
	id          int64
	providerID  int64
	accountID   int64
	sub         string
	displayName string
	createdAt   time.Time
}

// fakeFreqEdit mirrors a verge-core frequency edit row.
type fakeFreqEdit struct {
	action    string
	createdBy int64
}

// fakeChannel mirrors a channel row, secret included, so tests can assert the
// secret is stored but never surfaced through the render path.
type fakeChannel struct {
	id                     int64
	url                    string
	secret                 pgtype.Text
	drift, coverage, clock bool
	enabled                bool
	createdBy              int64
	createdAt, updatedAt   time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		accounts: map[int64]db.Account{}, byName: map[string]int64{}, nextID: 1,
		seedNextID: 1, exclNextID: 1, annoNextID: 1, vantageNextID: 1, chanNextID: 1,
		lookupNextID: 1, proposalNext: 1,
		sourceStates:      map[string]db.SourceState{},
		integrationStates: map[string]db.IntegrationState{},
		scans: []db.Scan{
			{ID: 1, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
			{ID: 2, Kind: "hot", Enabled: true, CadenceSeconds: 86400},
			// The cold Scan ships disabled with an empty scope list (ADR-0044).
			{ID: 3, Kind: "cold", Enabled: false, CadenceSeconds: 2592000},
		},
		obsNextID: 1, batchNextID: 1, scanNextID: 1, tokenNextID: 1,
		resetNextID: 1, recoveryNextID: 1, inviteNextID: 1,
		freqEdits:  map[int32]fakeFreqEdit{},
		coldScopes: map[int64]bool{},
	}
}

func (f *fakeStore) ListDispatchProgress(_ context.Context, limit int32) ([]db.ListDispatchProgressRow, error) {
	rows := f.dispatchProgress
	if int(limit) < len(rows) {
		rows = rows[:limit]
	}
	return rows, nil
}

func (f *fakeStore) ListJobsForDispatch(_ context.Context, dispatchID pgtype.Int8) ([]db.ListJobsForDispatchRow, error) {
	return f.jobsByDispatch[dispatchID.Int64], nil
}

func (f *fakeStore) ListColdScopeSeedIds(context.Context) ([]int64, error) {
	ids := make([]int64, 0, len(f.coldScopes))
	for id, in := range f.coldScopes {
		if in {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func (f *fakeStore) OptInColdScope(_ context.Context, arg db.OptInColdScopeParams) error {
	f.coldScopes[arg.SeedID] = true
	return nil
}

func (f *fakeStore) OptOutColdScope(_ context.Context, seedID int64) error {
	delete(f.coldScopes, seedID)
	return nil
}

// SyncColdScanEnabled mirrors the SQL: the cold Scan is enabled exactly while at
// least one Seed scope is opted in.
func (f *fakeStore) SyncColdScanEnabled(context.Context) error {
	enabled := len(f.coldScopes) > 0
	for i := range f.scans {
		if f.scans[i].Kind == "cold" {
			f.scans[i].Enabled = enabled
		}
	}
	return nil
}

func (f *fakeStore) ListVergeCoreFrequencyEditsWithAuthor(context.Context) ([]db.ListVergeCoreFrequencyEditsWithAuthorRow, error) {
	ports := make([]int, 0, len(f.freqEdits))
	for p := range f.freqEdits {
		ports = append(ports, int(p))
	}
	sort.Ints(ports)
	out := make([]db.ListVergeCoreFrequencyEditsWithAuthorRow, 0, len(ports))
	for i, p := range ports {
		e := f.freqEdits[int32(p)]
		out = append(out, db.ListVergeCoreFrequencyEditsWithAuthorRow{
			ID: int64(i + 1), Port: int32(p), Action: e.action, CreatedByUsername: "admin",
		})
	}
	return out, nil
}

func (f *fakeStore) UpsertVergeCoreFrequencyEdit(_ context.Context, arg db.UpsertVergeCoreFrequencyEditParams) error {
	f.freqEdits[arg.Port] = fakeFreqEdit{action: arg.Action, createdBy: arg.CreatedBy}
	return nil
}

func (f *fakeStore) DeleteVergeCoreFrequencyEdit(_ context.Context, port int32) error {
	delete(f.freqEdits, port)
	return nil
}

func (f *fakeStore) GetScanByKind(_ context.Context, kind string) (db.Scan, error) {
	for _, sc := range f.scans {
		if sc.Kind == kind {
			return sc, nil
		}
	}
	return db.Scan{}, pgx.ErrNoRows
}

func (f *fakeStore) ListScans(context.Context) ([]db.Scan, error) {
	if f.listScansErr != nil {
		return nil, f.listScansErr
	}
	return f.scans, nil
}

// TightestEnabledScanCadenceSeconds mirrors the MIN-over-enabled-Scans query the
// observation floor rests on (#208). newFakeStore seeds the dns Scan (daily), so
// the tightest bound in force is k*daily and the observation dial floors at 2 days.
func (f *fakeStore) TightestEnabledScanCadenceSeconds(context.Context) (int64, error) {
	var tightest int64
	for _, sc := range f.scans {
		if sc.Enabled && (tightest == 0 || sc.CadenceSeconds < tightest) {
			tightest = sc.CadenceSeconds
		}
	}
	return tightest, nil
}

func (f *fakeStore) RecordHeartbeat(context.Context) (db.Heartbeat, error) {
	return f.hb, f.hbErr
}

func (f *fakeStore) CountAccounts(context.Context) (int64, error) {
	return int64(len(f.accounts)), nil
}

func (f *fakeStore) CreateAccount(_ context.Context, arg db.CreateAccountParams) (db.Account, error) {
	if _, taken := f.byName[arg.Username]; taken {
		return db.Account{}, &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	}
	acct := db.Account{
		ID: f.nextID, Username: arg.Username, Role: arg.Role, PasswordHash: arg.PasswordHash,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.accounts[acct.ID] = acct
	f.byName[acct.Username] = acct.ID
	f.nextID++
	return acct, nil
}

func (f *fakeStore) GetAccountByUsername(_ context.Context, username string) (db.Account, error) {
	id, ok := f.byName[username]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return f.accounts[id], nil
}

func (f *fakeStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	acct, ok := f.accounts[id]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return acct, nil
}

func (f *fakeStore) SetTOTPSecret(_ context.Context, arg db.SetTOTPSecretParams) error {
	acct, ok := f.accounts[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	acct.TotpSecret = arg.TotpSecret
	acct.TotpEnabled = false
	f.accounts[arg.ID] = acct
	return nil
}

func (f *fakeStore) ConfirmTOTP(_ context.Context, id int64) error {
	acct, ok := f.accounts[id]
	if !ok || !acct.TotpSecret.Valid {
		return pgx.ErrNoRows
	}
	acct.TotpEnabled = true
	f.accounts[id] = acct
	return nil
}

func (f *fakeStore) DeleteAccount(_ context.Context, id int64) error {
	if _, ok := f.accounts[id]; !ok {
		return pgx.ErrNoRows
	}
	delete(f.accounts, id)
	for name, nid := range f.byName {
		if nid == id {
			delete(f.byName, name)
		}
	}
	return nil
}

func (f *fakeStore) ResetAccountTOTP(_ context.Context, id int64) error {
	acct, ok := f.accounts[id]
	if !ok {
		return pgx.ErrNoRows
	}
	acct.TotpSecret = pgtype.Text{}
	acct.TotpEnabled = false
	f.accounts[id] = acct
	return nil
}

func (f *fakeStore) CreateNameSeed(_ context.Context, arg db.CreateNameSeedParams) (db.Seed, error) {
	for _, s := range f.seeds {
		if s.Kind == "name" && s.NameDomain.String == arg.NameDomain.String {
			return db.Seed{}, &pgconn.PgError{Code: "23505", Message: "duplicate seed"}
		}
	}
	sd := db.Seed{
		ID: f.seedNextID, Kind: "name", NameDomain: arg.NameDomain, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.seeds = append(f.seeds, sd)
	f.seedNextID++
	return sd, nil
}

func (f *fakeStore) CreateAddressSeed(_ context.Context, arg db.CreateAddressSeedParams) (db.Seed, error) {
	for _, s := range f.seeds {
		if s.Kind == "address" && s.AddressCidr != nil && arg.AddressCidr != nil && s.AddressCidr.String() == arg.AddressCidr.String() {
			return db.Seed{}, &pgconn.PgError{Code: "23505", Message: "duplicate seed"}
		}
	}
	sd := db.Seed{
		ID: f.seedNextID, Kind: "address", AddressCidr: arg.AddressCidr, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.seeds = append(f.seeds, sd)
	f.seedNextID++
	return sd, nil
}

func (f *fakeStore) ListSeeds(context.Context) ([]db.ListSeedsRow, error) {
	rows := make([]db.ListSeedsRow, 0, len(f.seeds))
	// Newest first, mirroring the SQL ORDER BY created_at DESC, id DESC.
	for i := len(f.seeds) - 1; i >= 0; i-- {
		s := f.seeds[i]
		rows = append(rows, db.ListSeedsRow{
			ID: s.ID, Kind: s.Kind, NameDomain: s.NameDomain, AddressCidr: s.AddressCidr,
			CustodyExtension: s.CustodyExtension,
			CreatedBy:        s.CreatedBy, CreatedAt: s.CreatedAt,
			CreatedByUsername: f.accounts[s.CreatedBy].Username,
		})
	}
	return rows, nil
}

func (f *fakeStore) SetCustodyExtension(_ context.Context, arg db.SetCustodyExtensionParams) error {
	for i, s := range f.seeds {
		if s.ID == arg.ID && s.Kind == "name" {
			f.seeds[i].CustodyExtension = arg.CustodyExtension
			return nil
		}
	}
	return nil // a missing or address-scope row is a no-op, matching the SQL guard
}

func (f *fakeStore) CreateNameExclusion(_ context.Context, arg db.CreateNameExclusionParams) (db.Exclusion, error) {
	for _, e := range f.exclusions {
		if e.Kind == arg.Kind && e.Name.String == arg.Name.String {
			return db.Exclusion{}, &pgconn.PgError{Code: "23505", Message: "duplicate exclusion"}
		}
	}
	ex := db.Exclusion{
		ID: f.exclNextID, Kind: arg.Kind, Name: arg.Name, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.exclusions = append(f.exclusions, ex)
	f.exclNextID++
	return ex, nil
}

func (f *fakeStore) CreateAddressExclusion(_ context.Context, arg db.CreateAddressExclusionParams) (db.Exclusion, error) {
	for _, e := range f.exclusions {
		if e.Kind == "address" && e.AddressCidr != nil && arg.AddressCidr != nil && e.AddressCidr.String() == arg.AddressCidr.String() {
			return db.Exclusion{}, &pgconn.PgError{Code: "23505", Message: "duplicate exclusion"}
		}
	}
	ex := db.Exclusion{
		ID: f.exclNextID, Kind: "address", AddressCidr: arg.AddressCidr, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.exclusions = append(f.exclusions, ex)
	f.exclNextID++
	return ex, nil
}

func (f *fakeStore) ListExclusions(context.Context) ([]db.ListExclusionsRow, error) {
	rows := make([]db.ListExclusionsRow, 0, len(f.exclusions))
	// Newest first, mirroring the SQL ORDER BY created_at DESC, id DESC.
	for i := len(f.exclusions) - 1; i >= 0; i-- {
		e := f.exclusions[i]
		rows = append(rows, db.ListExclusionsRow{
			ID: e.ID, Kind: e.Kind, Name: e.Name, AddressCidr: e.AddressCidr,
			CreatedBy: e.CreatedBy, CreatedAt: e.CreatedAt,
			CreatedByUsername: f.accounts[e.CreatedBy].Username,
		})
	}
	return rows, nil
}

func (f *fakeStore) CreateVantage(_ context.Context, arg db.CreateVantageParams) (db.Vantage, error) {
	for _, v := range f.vantages {
		if v.Host.String == arg.Host && v.Port.Int32 == arg.Port && v.Username.String == arg.Username {
			return db.Vantage{}, &pgconn.PgError{Code: "23505", Message: "duplicate vantage"}
		}
	}
	// A provisioned prober carries its endpoint columns; the unified table leaves
	// them NULL only for the resolver-only local vantage, which is never created
	// through this path.
	v := db.Vantage{
		ID:           f.vantageNextID,
		Name:         arg.Name,
		Class:        "unverified",
		Host:         pgtype.Text{String: arg.Host, Valid: true},
		Port:         pgtype.Int4{Int32: arg.Port, Valid: true},
		Username:     pgtype.Text{String: arg.Username, Valid: true},
		Availability: pgtype.Text{String: "pending", Valid: true},
		CreatedBy:    pgtype.Int8{Int64: arg.CreatedBy, Valid: true},
		CreatedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v, nil
}

func (f *fakeStore) ListVantages(context.Context) ([]db.ListVantagesRow, error) {
	rows := make([]db.ListVantagesRow, 0, len(f.vantages))
	// Newest first, mirroring the SQL ORDER BY created_at DESC, id DESC. The web
	// list is scoped to provisioned probers (host set), so resolver-only rows are
	// skipped just as the query's WHERE host IS NOT NULL does.
	for i := len(f.vantages) - 1; i >= 0; i-- {
		v := f.vantages[i]
		if !v.Host.Valid {
			continue
		}
		rows = append(rows, db.ListVantagesRow{
			ID: v.ID, Name: v.Name, Class: v.Class, Resolver: v.Resolver,
			Host: v.Host, Port: v.Port, Username: v.Username,
			Availability: v.Availability, PublicKey: v.PublicKey, HostKey: v.HostKey,
			CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt,
			CreatedByUsername: f.accounts[v.CreatedBy.Int64].Username,
		})
	}
	return rows, nil
}

func (f *fakeStore) ListUnavailableVantages(context.Context) ([]db.ListUnavailableVantagesRow, error) {
	// Mirrors the query: every vantage marked unavailable, including the
	// resolver-only rows the prober list excludes, ordered by name.
	rows := make([]db.ListUnavailableVantagesRow, 0)
	for _, v := range f.vantages {
		if v.Availability.String != "unavailable" {
			continue
		}
		rows = append(rows, db.ListUnavailableVantagesRow{
			ID: v.ID, Name: v.Name, Class: v.Class, Resolver: v.Resolver, Availability: v.Availability,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows, nil
}

func (f *fakeStore) ListReachabilitySpansForExposure(context.Context) ([]db.ListReachabilitySpansForExposureRow, error) {
	return f.reachSpans, nil
}

func (f *fakeStore) DeleteExclusion(_ context.Context, id int64) error {
	for i, e := range f.exclusions {
		if e.ID == id {
			f.exclusions = append(f.exclusions[:i], f.exclusions[i+1:]...)
			return nil
		}
	}
	return nil // idempotent: a missing row is not an error
}

func (f *fakeStore) CreateAnnotation(_ context.Context, arg db.CreateAnnotationParams) (db.Annotation, error) {
	// The unique index on (subject_key, signal_name): a pair is declared once.
	for _, a := range f.annotations {
		if a.SubjectKey == arg.SubjectKey && a.SignalName == arg.SignalName {
			return db.Annotation{}, &pgconn.PgError{Code: "23505", Message: "duplicate annotation"}
		}
	}
	a := db.Annotation{
		ID: f.annoNextID, SubjectKey: arg.SubjectKey, SignalName: arg.SignalName,
		Reason: arg.Reason, DeclaredAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.annotations = append(f.annotations, a)
	f.annoNextID++
	return a, nil
}

func (f *fakeStore) ListAnnotations(context.Context) ([]db.Annotation, error) {
	rows := append([]db.Annotation(nil), f.annotations...)
	// ORDER BY signal_name, subject_key.
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SignalName != rows[j].SignalName {
			return rows[i].SignalName < rows[j].SignalName
		}
		return rows[i].SubjectKey < rows[j].SubjectKey
	})
	return rows, nil
}

func (f *fakeStore) DeleteAnnotation(_ context.Context, id int64) error {
	for i, a := range f.annotations {
		if a.ID == id {
			f.annotations = append(f.annotations[:i], f.annotations[i+1:]...)
			return nil
		}
	}
	return nil // idempotent: a missing row is not an error
}

func (f *fakeStore) InsertMessage(_ context.Context, arg db.InsertMessageParams) (db.Message, error) {
	if f.msgNextID == 0 {
		f.msgNextID = 1
	}
	m := db.Message{
		ID: f.msgNextID, Cause: arg.Cause, Class: arg.Class,
		SubjectKind: arg.SubjectKind, FiredAt: arg.FiredAt, Instant: arg.Instant,
		Census: arg.Census, Headline: arg.Headline,
	}
	f.msgNextID++
	f.messages = append(f.messages, m)
	return m, nil
}

func (f *fakeStore) ListMessages(context.Context) ([]db.Message, error) {
	out := make([]db.Message, len(f.messages))
	// Newest-first, mirroring ORDER BY id DESC.
	for i, m := range f.messages {
		out[len(f.messages)-1-i] = m
	}
	return out, nil
}

func (f *fakeStore) ListDeliveryOutcomes(context.Context) ([]db.ListDeliveryOutcomesRow, error) {
	return f.deliveryOutcomes, nil
}

func (f *fakeStore) readMarks(accountID int64) map[int64]bool {
	if f.messageRead == nil {
		f.messageRead = map[int64]map[int64]bool{}
	}
	set := f.messageRead[accountID]
	if set == nil {
		set = map[int64]bool{}
		f.messageRead[accountID] = set
	}
	return set
}

func (f *fakeStore) ListReadMessageIDs(_ context.Context, accountID int64) ([]int64, error) {
	set := f.readMarks(accountID)
	out := make([]int64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	return out, nil
}

func (f *fakeStore) CountUnreadMessages(_ context.Context, accountID int64) (int64, error) {
	set := f.readMarks(accountID)
	var n int64
	for _, m := range f.messages {
		if !set[m.ID] {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) MarkMessageRead(_ context.Context, arg db.MarkMessageReadParams) error {
	// Idempotent per account: a first mark stands (ON CONFLICT DO NOTHING).
	set := f.readMarks(arg.AccountID)
	if !set[arg.MessageID] {
		set[arg.MessageID] = true
	}
	return nil
}

func (f *fakeStore) MarkAllMessagesRead(_ context.Context, arg db.MarkAllMessagesReadParams) error {
	set := f.readMarks(arg.AccountID)
	for _, m := range f.messages {
		if !set[m.ID] {
			set[m.ID] = true
		}
	}
	return nil
}

func (f *fakeStore) PreviewExclusionWithdrawal(_ context.Context, _ db.PreviewExclusionWithdrawalParams) (db.PreviewExclusionWithdrawalRow, error) {
	return f.previewResult, nil
}

func (f *fakeStore) ListAccounts(context.Context) ([]db.ListAccountsRow, error) {
	// Insertion order (created_at ASC, id ASC) mirrors the SQL.
	ids := make([]int64, 0, len(f.accounts))
	for id := range f.accounts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	rows := make([]db.ListAccountsRow, 0, len(ids))
	for _, id := range ids {
		a := f.accounts[id]
		rows = append(rows, db.ListAccountsRow{
			ID: a.ID, Username: a.Username, Role: a.Role,
			TotpEnabled: a.TotpEnabled, CreatedAt: a.CreatedAt,
		})
	}
	return rows, nil
}

func (f *fakeStore) CountAdmins(context.Context) (int64, error) {
	var n int64
	for _, a := range f.accounts {
		if a.Role == roleAdmin {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) UpdateAccountRole(_ context.Context, arg db.UpdateAccountRoleParams) error {
	a, ok := f.accounts[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	a.Role = arg.Role
	f.accounts[arg.ID] = a
	return nil
}

func (f *fakeStore) CreateChannel(_ context.Context, arg db.CreateChannelParams) (int64, error) {
	c := fakeChannel{
		id: f.chanNextID, url: arg.Url, secret: arg.Secret,
		drift: arg.RouteDrift, coverage: arg.RouteCoverage, clock: arg.RouteClock,
		enabled: arg.Enabled, createdBy: arg.CreatedBy,
		createdAt: time.Now(), updatedAt: time.Now(),
	}
	f.channels = append(f.channels, c)
	f.chanNextID++
	return c.id, nil
}

func (f *fakeStore) ListChannels(context.Context) ([]db.ListChannelsRow, error) {
	rows := make([]db.ListChannelsRow, 0, len(f.channels))
	// Newest first, mirroring ORDER BY created_at DESC, id DESC.
	for i := len(f.channels) - 1; i >= 0; i-- {
		c := f.channels[i]
		rows = append(rows, db.ListChannelsRow{
			ID: c.id, Url: c.url, RouteDrift: c.drift, RouteCoverage: c.coverage,
			RouteClock: c.clock, Enabled: c.enabled, HasSecret: c.secret.Valid,
			CreatedBy:         c.createdBy,
			CreatedAt:         pgtype.Timestamptz{Time: c.createdAt, Valid: true},
			UpdatedAt:         pgtype.Timestamptz{Time: c.updatedAt, Valid: true},
			CreatedByUsername: f.accounts[c.createdBy].Username,
		})
	}
	return rows, nil
}

func (f *fakeStore) UpdateChannel(_ context.Context, arg db.UpdateChannelParams) error {
	for i := range f.channels {
		if f.channels[i].id == arg.ID {
			f.channels[i].url = arg.Url
			f.channels[i].drift = arg.RouteDrift
			f.channels[i].coverage = arg.RouteCoverage
			f.channels[i].clock = arg.RouteClock
			f.channels[i].enabled = arg.Enabled
			f.channels[i].updatedAt = time.Now()
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) SetChannelSecret(_ context.Context, arg db.SetChannelSecretParams) error {
	for i := range f.channels {
		if f.channels[i].id == arg.ID {
			f.channels[i].secret = arg.Secret
			f.channels[i].updatedAt = time.Now()
			return nil
		}
	}
	return pgx.ErrNoRows
}

func (f *fakeStore) DeleteChannel(_ context.Context, id int64) error {
	for i, c := range f.channels {
		if c.id == id {
			f.channels = append(f.channels[:i], f.channels[i+1:]...)
			return nil
		}
	}
	return nil // idempotent
}

func (f *fakeStore) GetRetentionSettings(context.Context) (db.GetRetentionSettingsRow, error) {
	return f.retention, nil
}

func (f *fakeStore) UpdateRetentionSettings(_ context.Context, arg db.UpdateRetentionSettingsParams) error {
	f.retention.ObservationCurrencyDays = arg.ObservationCurrencyDays
	f.retention.DispatchCadenceMultiple = arg.DispatchCadenceMultiple
	f.retention.UpdatedBy = arg.UpdatedBy
	f.retention.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	return nil
}

// --- Subjects reads --------------------------------------------------------

// ensureScan returns the id of the scan of the given kind, creating it once.
func (f *fakeStore) ensureScan(kind string) int64 {
	for _, sc := range f.scans {
		if sc.Kind == kind {
			return sc.ID
		}
	}
	sc := db.Scan{ID: f.scanNextID, Kind: kind, Enabled: true, CadenceSeconds: 86400}
	f.scans = append(f.scans, sc)
	f.scanNextID++
	return sc.ID
}

// freshBatch appends a completed batch of the given scan kind (creating the Scan
// once) and returns its id, so every seeded observation rides a real batch tied
// to an enabled Scan. That batch→Scan link is what the live-tier gate reads to
// find a timeline's covering cadence (#237): an observation with no such link has
// an undefined bound and is never live, so fixtures must carry one exactly as the
// measurement worker's observations do.
func (f *fakeStore) freshBatch(scanKind, batchKind string) int64 {
	scanID := f.ensureScan(scanKind)
	b := db.Batch{ID: f.batchNextID, ScanID: scanID, Kind: batchKind, Outcome: "completed"}
	f.batches = append(f.batches, b)
	f.batchNextID++
	return b.ID
}

// addResolution records a resolution observation for a Name in a fresh batch of
// the given scan kind, mirroring what the measurement worker writes. It is the
// only seam the Subjects tests need to populate the estate.
func (f *fakeStore) addResolution(t *testing.T, createdBy int64, name, scanKind string, at time.Time, value string) {
	t.Helper()
	b := f.freshBatch(scanKind, "resolution-walk")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "resolution", SubjectKind: "name",
		SubjectKey: name, Source: "resolver", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

// addAdmittedName records a CT admission for a Name in a fresh ct batch, mirroring
// what the crt.sh runner writes (ADR-0027, ADR-0106). It is the seam a Citation
// test uses to make a Name CT-admitted so its citation reconciles to the admission
// (ADR-0107).
func (f *fakeStore) addAdmittedName(t *testing.T, name string, at time.Time) {
	t.Helper()
	f.addAdmittedNameUnderSeed(t, name, f.coveringNameSeedID(name), at)
}

// addAdmittedNameUnderSeed admits a Name citing an explicit covering Seed id — the
// seed_id the runner records on the admitted_name row (ADR-0027). The seed id is
// what the Citation chain terminates at, so a test can admit a Name under one Seed
// while a longer-suffix Seed also covers it (#256).
func (f *fakeStore) addAdmittedNameUnderSeed(t *testing.T, name string, seedID int64, at time.Time) {
	t.Helper()
	b := f.freshBatch("ct", "ct")
	f.admitted = append(f.admitted, db.AdmittedName{
		ID: int64(len(f.admitted) + 1), Name: name, Source: "crtsh", SeedID: seedID, BatchID: b,
		CreatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

// coveringNameSeedID mirrors the runner's admission: a Name is admitted under the
// longest-suffix name Seed that covers it (ADR-0047).
func (f *fakeStore) coveringNameSeedID(name string) int64 {
	var best *db.Seed
	for i := range f.seeds {
		s := &f.seeds[i]
		if s.Kind != "name" || !s.NameDomain.Valid {
			continue
		}
		d := s.NameDomain.String
		if name == d || strings.HasSuffix(name, "."+d) {
			if best == nil || len(d) > len(best.NameDomain.String) {
				best = s
			}
		}
	}
	if best == nil {
		return 0
	}
	return best.ID
}

// liveObservations returns the live-tier subset of the observation corpus as of
// asOf — the fake twin of ListLiveObservationsForDerivation and
// retention.LiveOnly (#237, ADR-0041). It computes each timeline's tightest
// covering ENABLED-Scan cadence from the batch→Scan link, then keeps only rows
// whose age at asOf is within retention.FloorCadences of that bound. A timeline no
// enabled Scan covers has an undefined bound and yields no live row, exactly as
// the SQL `cover` JOIN drops it. Sharing retention.TierOf keeps this gate the same
// boundary the production reads and the pure retention tests use.
func (f *fakeStore) liveObservations(asOf time.Time) []db.Observation {
	enabledCadence := map[int64]int64{} // scan id -> cadence, enabled scans only
	for _, sc := range f.scans {
		if sc.Enabled {
			enabledCadence[sc.ID] = sc.CadenceSeconds
		}
	}
	batchCadence := map[int64]int64{} // batch id -> covering enabled cadence
	for _, b := range f.batches {
		if c, ok := enabledCadence[b.ScanID]; ok {
			batchCadence[b.ID] = c
		}
	}
	type timeline struct {
		subjectKey, facet, discriminator, source string
		vantage                                  int64
		vantageValid                             bool
	}
	keyOf := func(o db.Observation) timeline {
		return timeline{o.SubjectKey, o.Facet, o.Discriminator, o.Source, o.VantageID.Int64, o.VantageID.Valid}
	}
	tightest := map[timeline]int64{} // MIN covering cadence over a timeline's rows
	for _, o := range f.observations {
		c, ok := batchCadence[o.BatchID]
		if !ok {
			continue
		}
		k := keyOf(o)
		if cur, seen := tightest[k]; !seen || c < cur {
			tightest[k] = c
		}
	}
	out := make([]db.Observation, 0, len(f.observations))
	for _, o := range f.observations {
		bound, hasBound := retention.ObservationBoundSeconds(tightest[keyOf(o)])
		age := int64(asOf.Sub(o.ObservedAt.Time).Seconds())
		if retention.TierOf(age, bound, hasBound) == retention.Live {
			out = append(out, o)
		}
	}
	return out
}

// latestResolutionByName picks, per Name, its latest resolution observation —
// max observed_at, then max id — mirroring the DISTINCT ON in the SQL. It reads
// the caller-supplied observation slice (the live-tier subset), never f.observations
// directly, so the gate applies before the DISTINCT ON.
func (f *fakeStore) latestResolutionByName(obs []db.Observation) map[string]db.Observation {
	latest := map[string]db.Observation{}
	for _, o := range obs {
		if o.SubjectKind != "name" || o.Facet != "resolution" {
			continue
		}
		cur, ok := latest[o.SubjectKey]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[o.SubjectKey] = o
		}
	}
	return latest
}

func fakeResolutionOutcome(value []byte) string {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome
}

func (f *fakeStore) ListCurrentNameSubjects(_ context.Context, arg db.ListCurrentNameSubjectsParams) ([]db.ListCurrentNameSubjectsRow, error) {
	search := arg.Search
	latest := f.latestResolutionByName(f.liveObservations(arg.AsOf.Time))
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := []db.ListCurrentNameSubjectsRow{}
	for _, k := range keys {
		o := latest[k]
		if suppressesNameMembership(fakeResolutionOutcome(o.Value)) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(k), strings.ToLower(search)) {
			continue
		}
		rows = append(rows, db.ListCurrentNameSubjectsRow{
			SubjectKey: k, Value: o.Value, ObservedAt: o.ObservedAt,
		})
	}
	return rows, nil
}

type fakeZoneFile struct {
	seedID     int64
	suppliedAt time.Time
	content    string
	uploadedBy int64
}

func (f *fakeStore) CreateZoneFile(_ context.Context, arg db.CreateZoneFileParams) (db.CreateZoneFileRow, error) {
	f.zoneFiles = append(f.zoneFiles, fakeZoneFile{
		seedID: arg.SeedID, suppliedAt: arg.SuppliedAt.Time, content: arg.Content, uploadedBy: arg.UploadedBy,
	})
	f.zoneNextID++
	return db.CreateZoneFileRow{ID: f.zoneNextID, SuppliedAt: arg.SuppliedAt}, nil
}

func (f *fakeStore) ListZoneFileStatus(context.Context) ([]db.ListZoneFileStatusRow, error) {
	// Latest supply per seed, mirroring the SQL DISTINCT ON.
	latest := map[int64]fakeZoneFile{}
	for _, z := range f.zoneFiles {
		cur, ok := latest[z.seedID]
		if !ok || !z.suppliedAt.Before(cur.suppliedAt) {
			latest[z.seedID] = z
		}
	}
	rows := make([]db.ListZoneFileStatusRow, 0, len(latest))
	for _, s := range f.seeds {
		if s.Kind != "name" {
			continue
		}
		z, ok := latest[s.ID]
		if !ok {
			continue
		}
		rows = append(rows, db.ListZoneFileStatusRow{
			SeedID:             s.ID,
			NameDomain:         s.NameDomain,
			SuppliedAt:         pgtype.Timestamptz{Time: z.suppliedAt, Valid: true},
			UploadedByUsername: f.accounts[z.uploadedBy].Username,
			ContentBytes:       int64(len(z.content)),
		})
	}
	return rows, nil
}

// fakeFacetVector mirrors queue.facetVector: reachability folds under the single
// connect-outcome leaf; resolution and dns-record under the two membership leaves
// jointly (ADR-0086). The fake derives spans so the web tests stay hermetic.
func fakeFacetVector(facet string) drift.Vector {
	if facet == connectoutcome.FacetReachability {
		return drift.NewVector(drift.Component{Leaf: connectoutcome.Kind, Version: connectoutcome.Version})
	}
	if facet == httpexchange.FacetHTTPIdentity {
		return drift.NewVector(drift.Component{Leaf: httpexchange.Kind, Version: httpexchange.Version})
	}
	return drift.NewVector(
		drift.Component{Leaf: "resolution-walk", Version: resolutionwalk.Version},
		drift.Component{Leaf: "wildcard-discrimination", Version: wildcarddiscrim.Version},
	)
}

// ListAllOpenSpans folds every observation the fake holds into Span rows using
// the real drift.Fold — exactly as ListSpansForSubject does per subject — and
// returns the OPEN ones across the whole estate, ordered by (kind, key, facet,
// discriminator) as the production query is. It is the Inventory axis read (#243):
// each open span is the value a timeline currently holds.
func (f *fakeStore) ListAllOpenSpans(_ context.Context) ([]db.ListAllOpenSpansRow, error) {
	type tlkey struct{ kind, key, facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	for _, o := range f.observations {
		k := tlkey{kind: o.SubjectKind, key: o.SubjectKey, facet: o.Facet, discriminator: o.Discriminator, source: o.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		gap := o.Facet == "resolution" && fakeResolutionOutcome(o.Value) == "Gap"
		byKey[k] = append(byKey[k], drift.Reading{
			Value: string(o.Value), IsGap: gap, Vector: fakeFacetVector(o.Facet), ObservedAt: o.ObservedAt.Time,
		})
	}

	sort.Slice(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.key != b.key {
			return a.key < b.key
		}
		if a.facet != b.facet {
			return a.facet < b.facet
		}
		if a.discriminator != b.discriminator {
			return a.discriminator < b.discriminator
		}
		return a.source < b.source
	})

	rows := []db.ListAllOpenSpansRow{}
	var id int64
	for _, k := range order {
		derivation, _ := json.Marshal(fakeFacetVector(k.facet))
		key := drift.TimelineKey{
			SubjectKind: k.kind, SubjectKey: k.key,
			Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
		}
		for _, s := range drift.Fold(key, byKey[k]) {
			if !s.Open() {
				continue
			}
			id++
			rows = append(rows, db.ListAllOpenSpansRow{
				ID: id, SubjectKind: k.kind, SubjectKey: k.key,
				Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
				Value: []byte(s.Value), IsGap: s.IsGap, Derivation: derivation,
				OpenedAt: pgtype.Timestamptz{Time: s.OpenedAt, Valid: true},
			})
		}
	}
	return rows, nil
}

// ListSpansForSubject folds the fake's observations for one subject into Span
// rows using the real drift.Fold, so the drill-down test exercises the same
// open/close logic the worker's ingest does. It is facet-generic: a `service`
// subject's `reachability` observations fold exactly as a `name`'s resolution
// ones do. The production store reads persisted spans; the fake derives them.
func (f *fakeStore) ListSpansForSubject(_ context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error) {
	type tlkey struct{ facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	for _, o := range f.observations {
		if o.SubjectKind != arg.SubjectKind || o.SubjectKey != arg.SubjectKey {
			continue
		}
		k := tlkey{facet: o.Facet, discriminator: o.Discriminator, source: o.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		gap := o.Facet == "resolution" && fakeResolutionOutcome(o.Value) == "Gap"
		byKey[k] = append(byKey[k], drift.Reading{
			Value: string(o.Value), IsGap: gap, Vector: fakeFacetVector(o.Facet), ObservedAt: o.ObservedAt.Time,
		})
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].facet != order[j].facet {
			return order[i].facet < order[j].facet
		}
		return order[i].discriminator < order[j].discriminator
	})

	rows := []db.ListSpansForSubjectRow{}
	var id int64
	for _, k := range order {
		derivation, _ := json.Marshal(fakeFacetVector(k.facet))
		key := drift.TimelineKey{
			SubjectKind: arg.SubjectKind, SubjectKey: arg.SubjectKey,
			Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
		}
		for _, s := range drift.Fold(key, byKey[k]) {
			id++
			row := db.ListSpansForSubjectRow{
				ID: id, SubjectKind: arg.SubjectKind, SubjectKey: arg.SubjectKey,
				Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
				Value: []byte(s.Value), IsGap: s.IsGap, Derivation: derivation,
				OpenedAt: pgtype.Timestamptz{Time: s.OpenedAt, Valid: true},
			}
			if !s.Open() {
				row.ClosedAt = pgtype.Timestamptz{Time: s.ClosedAt, Valid: true}
			}
			if s.Reason != "" {
				row.ClosureReason = pgtype.Text{String: string(s.Reason), Valid: true}
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// fakeBatchByID returns the batch with the given id, or a zero batch where none is
// held (an observation always cites a batch the fake created, so this resolves).
func (f *fakeStore) fakeBatchByID(id int64) db.Batch {
	for _, b := range f.batches {
		if b.ID == id {
			return b
		}
	}
	return db.Batch{}
}

// ListRecentDriftEvents folds every observation into Span timelines with the real
// drift.Fold — the same open/close logic the worker's ingest runs — and emits one
// 'opened' event per span, attributing it to the batch of the observation at the
// span's opened instant (the fake's observations carry their batch id, exactly as the
// production span carries opened_batch_id after ADR-0111). Each opened event carries
// its predecessor span so the handler classifies appeared vs changed and builds the
// diff. A reasoned close would emit a 'closed' event too, but the fold sets no reason
// (withdrawal persistence is unwired), so none arise here — matching production.
// Rows are ordered newest batch first, then by timeline, as the production query is.
func (f *fakeStore) ListRecentDriftEvents(_ context.Context, arg db.ListRecentDriftEventsParams) ([]db.ListRecentDriftEventsRow, error) {
	since := arg.Since
	type tlkey struct{ kind, key, facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	obsBatch := map[tlkey]map[int64]int64{} // per timeline: observedAt UnixNano -> batch id
	for _, o := range f.observations {
		k := tlkey{kind: o.SubjectKind, key: o.SubjectKey, facet: o.Facet, discriminator: o.Discriminator, source: o.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
			obsBatch[k] = map[int64]int64{}
		}
		gap := o.Facet == "resolution" && fakeResolutionOutcome(o.Value) == "Gap"
		byKey[k] = append(byKey[k], drift.Reading{
			Value: string(o.Value), IsGap: gap, Vector: fakeFacetVector(o.Facet), ObservedAt: o.ObservedAt.Time,
		})
		obsBatch[k][o.ObservedAt.Time.UnixNano()] = o.BatchID
	}

	rows := []db.ListRecentDriftEventsRow{}
	for _, k := range order {
		derivation, _ := json.Marshal(fakeFacetVector(k.facet))
		key := drift.TimelineKey{
			SubjectKind: k.kind, SubjectKey: k.key,
			Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
		}
		spans := drift.Fold(key, byKey[k])
		for i, s := range spans {
			if !since.Time.IsZero() && s.OpenedAt.Before(since.Time) {
				continue
			}
			bID := obsBatch[k][s.OpenedAt.UnixNano()]
			b := f.fakeBatchByID(bID)
			row := db.ListRecentDriftEventsRow{
				Role:          "opened",
				BatchID:       bID,
				BatchKind:     b.Kind,
				BatchAt:       pgtype.Timestamptz{Time: s.OpenedAt, Valid: true},
				RecordedScope: []byte(`{}`),
				SubjectKind:   k.kind, SubjectKey: k.key, Facet: k.facet, Discriminator: k.discriminator,
				Value: []byte(s.Value), IsGap: s.IsGap, Derivation: derivation,
				OpenedAt: pgtype.Timestamptz{Time: s.OpenedAt, Valid: true},
			}
			if !s.Open() {
				row.ClosedAt = pgtype.Timestamptz{Time: s.ClosedAt, Valid: true}
			}
			if i > 0 {
				prev := spans[i-1]
				row.PrevValue = []byte(prev.Value)
				row.PrevDerivation = derivation
				if !prev.ClosedAt.IsZero() {
					row.PrevClosedAt = pgtype.Timestamptz{Time: prev.ClosedAt, Valid: true}
				}
				if prev.Reason != "" {
					row.PrevClosureReason = pgtype.Text{String: string(prev.Reason), Valid: true}
				}
			}
			rows = append(rows, row)
		}
	}

	// Newest batch first, then by timeline — same order as the production query, so a
	// run of consecutive rows sharing a batch id is one group for buildDriftFeed.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if !a.BatchAt.Time.Equal(b.BatchAt.Time) {
			return a.BatchAt.Time.After(b.BatchAt.Time)
		}
		if a.BatchID != b.BatchID {
			return a.BatchID > b.BatchID
		}
		if a.SubjectKind != b.SubjectKind {
			return a.SubjectKind < b.SubjectKind
		}
		if a.SubjectKey != b.SubjectKey {
			return a.SubjectKey < b.SubjectKey
		}
		return a.Facet < b.Facet
	})
	// Honor the feed cap: the newest events survive (rows are already newest-first).
	if arg.MaxEvents > 0 && int32(len(rows)) > arg.MaxEvents {
		rows = rows[:arg.MaxEvents]
	}
	return rows, nil
}

// addReachability records a reachability observation for a Service in a fresh
// batch — the connect-outcome leaf's output the hot Scan writes. It is the seam
// the Service drill-down tests populate.
func (f *fakeStore) addReachability(t *testing.T, serviceKey string, at time.Time, value string) {
	t.Helper()
	scanID := f.ensureScan("hot")
	b := db.Batch{ID: f.batchNextID, ScanID: scanID, Kind: "connect-outcome", Outcome: "completed"}
	f.batches = append(f.batches, b)
	f.batchNextID++
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b.ID, Facet: "reachability", SubjectKind: "service",
		SubjectKey: serviceKey, Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

func (f *fakeStore) latestReachabilityByService(obs []db.Observation) map[string]db.Observation {
	latest := map[string]db.Observation{}
	for _, o := range obs {
		if o.SubjectKind != "service" || o.Facet != "reachability" {
			continue
		}
		cur, ok := latest[o.SubjectKey]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[o.SubjectKey] = o
		}
	}
	return latest
}

func (f *fakeStore) ListCurrentServiceSubjects(_ context.Context, arg db.ListCurrentServiceSubjectsParams) ([]db.ListCurrentServiceSubjectsRow, error) {
	search := arg.Search
	latest := f.latestReachabilityByService(f.liveObservations(arg.AsOf.Time))
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := []db.ListCurrentServiceSubjectsRow{}
	for _, k := range keys {
		if search != "" && !strings.Contains(strings.ToLower(k), strings.ToLower(search)) {
			continue
		}
		o := latest[k]
		rows = append(rows, db.ListCurrentServiceSubjectsRow{
			SubjectKey: k, Value: o.Value, ObservedAt: o.ObservedAt,
		})
	}
	return rows, nil
}

func (f *fakeStore) GetServiceSubject(_ context.Context, arg db.GetServiceSubjectParams) (db.GetServiceSubjectRow, error) {
	key := arg.SubjectKey
	o, ok := f.latestReachabilityByService(f.liveObservations(arg.AsOf.Time))[key]
	if !ok {
		return db.GetServiceSubjectRow{}, pgx.ErrNoRows
	}
	return db.GetServiceSubjectRow{SubjectKey: key, Value: o.Value, ObservedAt: o.ObservedAt}, nil
}

// addHTTPIdentity records an http-identity observation for an Endpoint in a fresh
// batch — the http-exchange leaf's output the hot Scan writes (#198). It is the
// seam the Endpoint drill-down tests populate.
func (f *fakeStore) addHTTPIdentity(t *testing.T, endpointKey string, at time.Time, value string) {
	t.Helper()
	scanID := f.ensureScan("hot")
	b := db.Batch{ID: f.batchNextID, ScanID: scanID, Kind: "http-exchange", Outcome: "completed"}
	f.batches = append(f.batches, b)
	f.batchNextID++
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b.ID, Facet: "http-identity", SubjectKind: "endpoint",
		SubjectKey: endpointKey, Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

func (f *fakeStore) latestHTTPIdentityByEndpoint(obs []db.Observation) map[string]db.Observation {
	latest := map[string]db.Observation{}
	for _, o := range obs {
		if o.SubjectKind != "endpoint" || o.Facet != "http-identity" {
			continue
		}
		cur, ok := latest[o.SubjectKey]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[o.SubjectKey] = o
		}
	}
	return latest
}

func (f *fakeStore) ListCurrentEndpointSubjects(_ context.Context, arg db.ListCurrentEndpointSubjectsParams) ([]db.ListCurrentEndpointSubjectsRow, error) {
	search := arg.Search
	latest := f.latestHTTPIdentityByEndpoint(f.liveObservations(arg.AsOf.Time))
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := []db.ListCurrentEndpointSubjectsRow{}
	for _, k := range keys {
		if search != "" && !strings.Contains(strings.ToLower(k), strings.ToLower(search)) {
			continue
		}
		o := latest[k]
		rows = append(rows, db.ListCurrentEndpointSubjectsRow{
			SubjectKey: k, Value: o.Value, ObservedAt: o.ObservedAt,
		})
	}
	return rows, nil
}

func (f *fakeStore) GetEndpointSubject(_ context.Context, arg db.GetEndpointSubjectParams) (db.GetEndpointSubjectRow, error) {
	key := arg.SubjectKey
	o, ok := f.latestHTTPIdentityByEndpoint(f.liveObservations(arg.AsOf.Time))[key]
	if !ok {
		return db.GetEndpointSubjectRow{}, pgx.ErrNoRows
	}
	return db.GetEndpointSubjectRow{SubjectKey: key, Value: o.Value, ObservedAt: o.ObservedAt}, nil
}

// addClassReachability records a reachability observation for a Service at a
// Vantage of the given class — the sensitive-port rule reads the internet-class
// leg, so its census needs the class join the plain Service read does not (#203).
func (f *fakeStore) addClassReachability(t *testing.T, serviceKey, class string, at time.Time, value string) {
	t.Helper()
	vid := f.vantageForClass(class)
	b := f.freshBatch("hot", "connect-outcome")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "reachability", SubjectKind: "service",
		SubjectKey: serviceKey, VantageID: pgtype.Int8{Int64: vid, Valid: true},
		Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

// reachClassKey is one (Service, Vantage class) reachability leg.
type reachClassKey struct{ svc, class string }

// currentReachByClass folds the fake's reachability observations into the current
// value per (Service, class) — the span corpus's current-state read, which is NOT
// live-tier-gated (spans are the already-derived timeline, ADR-0041), so it folds
// every observation rather than only the live ones. is_gap is derived from the
// value's outcome, exactly as the real fold's isGapValue reads it (ADR-0104).
func (f *fakeStore) currentReachByClass() map[reachClassKey]db.Observation {
	classOf := map[int64]string{}
	for _, v := range f.vantages {
		classOf[v.ID] = v.Class
	}
	latest := map[reachClassKey]db.Observation{}
	for _, o := range f.observations {
		if o.SubjectKind != "service" || o.Facet != "reachability" || !o.VantageID.Valid {
			continue
		}
		class, ok := classOf[o.VantageID.Int64]
		if !ok {
			continue
		}
		k := reachClassKey{o.SubjectKey, class}
		cur, ok := latest[k]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[k] = o
		}
	}
	return latest
}

func reachOutcomeIsGap(value []byte) bool {
	var v struct {
		Outcome string `json:"outcome"`
	}
	_ = json.Unmarshal(value, &v)
	return v.Outcome == "gap"
}

func (f *fakeStore) ListServiceReachabilitySpansByClass(_ context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error) {
	rows := []db.ListServiceReachabilitySpansByClassRow{}
	for k, o := range f.currentReachByClass() {
		rows = append(rows, db.ListServiceReachabilitySpansByClassRow{
			SubjectKey: k.svc, Class: k.class, Value: o.Value, IsGap: reachOutcomeIsGap(o.Value),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SubjectKey != rows[j].SubjectKey {
			return rows[i].SubjectKey < rows[j].SubjectKey
		}
		return rows[i].Class < rows[j].Class
	})
	return rows, nil
}

func (f *fakeStore) ListBlanketedReachServices(_ context.Context) ([]string, error) {
	seen := map[string]struct{}{}
	for k, o := range f.currentReachByClass() {
		if reachOutcomeIsGap(o.Value) {
			seen[k.svc] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for svc := range seen {
		out = append(out, svc)
	}
	sort.Strings(out)
	return out, nil
}

// addCertificate records a certificate observation for an Endpoint — the
// tls-handshake step's output (#197), the value the certificate rules read (#203).
func (f *fakeStore) addCertificate(t *testing.T, endpointKey string, at time.Time, value string) {
	t.Helper()
	b := f.freshBatch("hot", "tls-handshake")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "certificate", SubjectKind: "endpoint",
		SubjectKey: endpointKey, Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

func (f *fakeStore) ListEndpointCertificates(_ context.Context, arg db.ListEndpointCertificatesParams) ([]db.ListEndpointCertificatesRow, error) {
	latest := map[string]db.Observation{}
	for _, o := range f.liveObservations(arg.AsOf.Time) {
		if o.SubjectKind != "endpoint" || o.Facet != "certificate" {
			continue
		}
		cur, ok := latest[o.SubjectKey]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[o.SubjectKey] = o
		}
	}
	rows := []db.ListEndpointCertificatesRow{}
	for k, o := range latest {
		rows = append(rows, db.ListEndpointCertificatesRow{SubjectKey: k, Value: o.Value})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SubjectKey < rows[j].SubjectKey })
	return rows, nil
}

func (f *fakeStore) FindNameCitingAddress(_ context.Context, arg db.FindNameCitingAddressParams) (db.FindNameCitingAddressRow, error) {
	address := arg.Address
	// The earliest current resolution whose Resolved answer names the address.
	var best *db.FindNameCitingAddressRow
	for name, o := range f.latestResolutionByName(f.liveObservations(arg.AsOf.Time)) {
		if fakeResolutionOutcome(o.Value) != "Resolved" {
			continue
		}
		var v struct {
			Addresses []string `json:"addresses"`
		}
		_ = json.Unmarshal(o.Value, &v)
		for _, a := range v.Addresses {
			if a == address {
				cand := db.FindNameCitingAddressRow{SubjectKey: name, ObservedAt: o.ObservedAt}
				if best == nil || cand.ObservedAt.Time.Before(best.ObservedAt.Time) {
					best = &cand
				}
			}
		}
	}
	if best == nil {
		return db.FindNameCitingAddressRow{}, pgx.ErrNoRows
	}
	return *best, nil
}

func (f *fakeStore) FindCoveringAddressSeed(_ context.Context, address netip.Addr) (db.FindCoveringAddressSeedRow, error) {
	var best *db.FindCoveringAddressSeedRow
	var bestBits int
	for _, s := range f.seeds {
		if s.Kind != "address" || s.AddressCidr == nil {
			continue
		}
		if s.AddressCidr.Contains(address) {
			if best == nil || s.AddressCidr.Bits() > bestBits {
				row := db.FindCoveringAddressSeedRow{
					ID: s.ID, AddressCidr: s.AddressCidr, CreatedAt: s.CreatedAt,
					CreatedByUsername: f.accounts[s.CreatedBy].Username,
				}
				best = &row
				bestBits = s.AddressCidr.Bits()
			}
		}
	}
	if best == nil {
		return db.FindCoveringAddressSeedRow{}, pgx.ErrNoRows
	}
	return *best, nil
}

// addVantageClass registers a resolver-only vantage of the given class and
// returns its id, so a resolution observation can be tied to a Vantage class the
// Signals reads join against.
func (f *fakeStore) addVantageClass(class string) int64 {
	v := db.Vantage{ID: f.vantageNextID, Name: class + "-resolver", Class: class}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v.ID
}

// addClassResolution records a resolution observation at a Vantage of the given
// class — the Signals reads need the class join the plain Subjects reads do not.
func (f *fakeStore) addClassResolution(t *testing.T, name, class string, at time.Time, value string) {
	t.Helper()
	vid := f.vantageForClass(class)
	b := f.freshBatch("dns", "resolution-walk")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "resolution", SubjectKind: "name",
		SubjectKey: name, VantageID: pgtype.Int8{Int64: vid, Valid: true},
		Source: "resolver", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

// addDNSRecord records a dns-record observation for a Name on one qtype
// discriminator (CNAME, NS, …) — the CNAME target and NS Lame verdict the rules
// read.
func (f *fakeStore) addDNSRecord(t *testing.T, name, discriminator string, at time.Time, value string) {
	t.Helper()
	b := f.freshBatch("dns", "resolution-walk")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "dns-record", SubjectKind: "name",
		SubjectKey: name, Discriminator: discriminator, Source: "resolver",
		Value: []byte(value), ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

func (f *fakeStore) vantageForClass(class string) int64 {
	for _, v := range f.vantages {
		if v.Class == class && !v.Host.Valid {
			return v.ID
		}
	}
	return f.addVantageClass(class)
}

func (f *fakeStore) ListNameResolutionsByClass(_ context.Context, arg db.ListNameResolutionsByClassParams) ([]db.ListNameResolutionsByClassRow, error) {
	classOf := map[int64]string{}
	for _, v := range f.vantages {
		classOf[v.ID] = v.Class
	}
	type key struct{ name, class string }
	latest := map[key]db.Observation{}
	for _, o := range f.liveObservations(arg.AsOf.Time) {
		if o.SubjectKind != "name" || o.Facet != "resolution" || !o.VantageID.Valid {
			continue
		}
		class, ok := classOf[o.VantageID.Int64]
		if !ok {
			continue
		}
		k := key{o.SubjectKey, class}
		cur, ok := latest[k]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[k] = o
		}
	}
	rows := []db.ListNameResolutionsByClassRow{}
	for k, o := range latest {
		rows = append(rows, db.ListNameResolutionsByClassRow{SubjectKey: k.name, Class: k.class, Value: o.Value})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SubjectKey != rows[j].SubjectKey {
			return rows[i].SubjectKey < rows[j].SubjectKey
		}
		return rows[i].Class < rows[j].Class
	})
	return rows, nil
}

func (f *fakeStore) ListNameDNSRecords(_ context.Context, arg db.ListNameDNSRecordsParams) ([]db.ListNameDNSRecordsRow, error) {
	type key struct{ name, disc string }
	latest := map[key]db.Observation{}
	for _, o := range f.liveObservations(arg.AsOf.Time) {
		if o.SubjectKind != "name" || o.Facet != "dns-record" {
			continue
		}
		k := key{o.SubjectKey, o.Discriminator}
		cur, ok := latest[k]
		if !ok || o.ObservedAt.Time.After(cur.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(cur.ObservedAt.Time) && o.ID > cur.ID) {
			latest[k] = o
		}
	}
	rows := []db.ListNameDNSRecordsRow{}
	for k, o := range latest {
		rows = append(rows, db.ListNameDNSRecordsRow{SubjectKey: k.name, Discriminator: k.disc, Value: o.Value})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SubjectKey != rows[j].SubjectKey {
			return rows[i].SubjectKey < rows[j].SubjectKey
		}
		return rows[i].Discriminator < rows[j].Discriminator
	})
	return rows, nil
}

func (f *fakeStore) ListZoneDeclarations(context.Context) ([]db.ListZoneDeclarationsRow, error) {
	latest := map[int64]fakeZoneFile{}
	for _, z := range f.zoneFiles {
		cur, ok := latest[z.seedID]
		if !ok || !z.suppliedAt.Before(cur.suppliedAt) {
			latest[z.seedID] = z
		}
	}
	rows := []db.ListZoneDeclarationsRow{}
	for _, s := range f.seeds {
		if s.Kind != "name" || !s.NameDomain.Valid {
			continue
		}
		z, ok := latest[s.ID]
		if !ok {
			continue
		}
		rows = append(rows, db.ListZoneDeclarationsRow{NameDomain: s.NameDomain, Content: z.content})
	}
	return rows, nil
}

func (f *fakeStore) GetNameSubject(_ context.Context, arg db.GetNameSubjectParams) (db.GetNameSubjectRow, error) {
	key := arg.SubjectKey
	o, ok := f.latestResolutionByName(f.liveObservations(arg.AsOf.Time))[key]
	if !ok {
		return db.GetNameSubjectRow{}, pgx.ErrNoRows
	}
	return db.GetNameSubjectRow{SubjectKey: key, Value: o.Value, ObservedAt: o.ObservedAt}, nil
}

// scanFor resolves a batch id to its (scan id, scan kind) — the batch→Scan hop
// both GetNameCitation hops need to name the introducing Scan.
func (f *fakeStore) scanFor(batchID int64) (int64, string) {
	var scanID int64
	for _, b := range f.batches {
		if b.ID == batchID {
			scanID = b.ScanID
		}
	}
	var scanKind string
	for _, sc := range f.scans {
		if sc.ID == scanID {
			scanKind = sc.Kind
		}
	}
	return scanID, scanKind
}

func (f *fakeStore) GetNameCitation(_ context.Context, arg db.GetNameCitationParams) (db.GetNameCitationRow, error) {
	key := arg.SubjectKey

	// ADR-0107: the admission wins. The latest admitted_name for the key is what
	// introduced the Name, so it is the citation whether or not a resolution has
	// since measured it.
	var admission *db.AdmittedName
	for i := range f.admitted {
		a := &f.admitted[i]
		if a.Name != key {
			continue
		}
		if admission == nil || a.ID > admission.ID {
			admission = a
		}
	}
	if admission != nil {
		scanID, scanKind := f.scanFor(admission.BatchID)
		return db.GetNameCitationRow{
			ObservedAt: admission.CreatedAt, Source: admission.Source,
			BatchID: admission.BatchID, ScanID: scanID, ScanKind: scanKind,
			// admitted_name.seed_id is NOT NULL in the schema — a real FK on every
			// admission. Mark the column NULL only for the degenerate fixture with no
			// covering Seed (id 0), so the fake never claims a valid seed_id of 0, a
			// state production forbids.
			SeedID:  pgtype.Int8{Int64: admission.SeedID, Valid: admission.SeedID != 0},
			HopKind: hopKindAdmission,
		}, nil
	}

	live := f.liveObservations(arg.AsOf.Time)
	var best *db.Observation
	for i := range live {
		o := &live[i]
		if o.SubjectKind != "name" || o.Facet != "resolution" || o.SubjectKey != key {
			continue
		}
		if best == nil || o.ObservedAt.Time.Before(best.ObservedAt.Time) ||
			(o.ObservedAt.Time.Equal(best.ObservedAt.Time) && o.ID < best.ID) {
			best = o
		}
	}
	if best == nil {
		return db.GetNameCitationRow{}, pgx.ErrNoRows
	}
	scanID, scanKind := f.scanFor(best.BatchID)
	return db.GetNameCitationRow{
		ObservedAt: best.ObservedAt, Source: best.Source,
		VantageID: best.VantageID, BatchID: best.BatchID, ScanID: scanID, ScanKind: scanKind,
		HopKind: hopKindObservation,
	}, nil
}

func (f *fakeStore) FindCoveringNameSeed(_ context.Context, name string) (db.FindCoveringNameSeedRow, error) {
	var best *db.Seed
	for i := range f.seeds {
		s := &f.seeds[i]
		if s.Kind != "name" || !s.NameDomain.Valid {
			continue
		}
		d := s.NameDomain.String
		if name == d || strings.HasSuffix(name, "."+d) {
			if best == nil || len(d) > len(best.NameDomain.String) {
				best = s
			}
		}
	}
	if best == nil {
		return db.FindCoveringNameSeedRow{}, pgx.ErrNoRows
	}
	return db.FindCoveringNameSeedRow{
		ID: best.ID, NameDomain: best.NameDomain, CreatedAt: best.CreatedAt,
		CreatedByUsername: f.accounts[best.CreatedBy].Username,
	}, nil
}

// FindNameSeedByID returns the name Seed with the given id (ADR-0027, #256) — the
// terminating hop a CT admission's Citation chain reads straight from the
// admitted_name row rather than re-deriving by suffix.
func (f *fakeStore) FindNameSeedByID(_ context.Context, seedID int64) (db.FindNameSeedByIDRow, error) {
	for i := range f.seeds {
		s := &f.seeds[i]
		if s.Kind != "name" || s.ID != seedID {
			continue
		}
		return db.FindNameSeedByIDRow{
			ID: s.ID, NameDomain: s.NameDomain, CreatedAt: s.CreatedAt,
			CreatedByUsername: f.accounts[s.CreatedBy].Username,
		}, nil
	}
	return db.FindNameSeedByIDRow{}, pgx.ErrNoRows
}

func (f *fakeStore) GetZoneCadenceSeconds(context.Context) (int64, error) {
	if f.zoneCadence == 0 {
		return 2592000, nil // the shipped monthly default
	}
	return f.zoneCadence, nil
}

func (f *fakeStore) SetZoneCadenceSeconds(_ context.Context, cadenceSeconds int64) error {
	f.zoneCadence = cadenceSeconds
	return nil
}

func (f *fakeStore) CreateProposerLookup(_ context.Context, arg db.CreateProposerLookupParams) (db.ProposerLookup, error) {
	l := db.ProposerLookup{
		ID: f.lookupNextID, Query: arg.Query, CreatedBy: arg.CreatedBy,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.lookups = append(f.lookups, l)
	f.lookupNextID++
	return l, nil
}

func (f *fakeStore) CreateProposal(_ context.Context, arg db.CreateProposalParams) (db.Proposal, error) {
	p := db.Proposal{
		ID: f.proposalNext, LookupID: arg.LookupID, SourceSlug: arg.SourceSlug,
		RecordKind: arg.RecordKind, AddressCidr: arg.AddressCidr, OrgName: arg.OrgName,
		Status:    "pending",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.proposals = append(f.proposals, p)
	f.proposalNext++
	return p, nil
}

func (f *fakeStore) ListPendingProposals(context.Context) ([]db.ListPendingProposalsRow, error) {
	lookupByID := map[int64]db.ProposerLookup{}
	for _, l := range f.lookups {
		lookupByID[l.ID] = l
	}
	rows := []db.ListPendingProposalsRow{}
	for _, p := range f.proposals {
		if p.Status != "pending" {
			continue
		}
		l := lookupByID[p.LookupID]
		rows = append(rows, db.ListPendingProposalsRow{
			ID: p.ID, LookupID: p.LookupID, SourceSlug: p.SourceSlug,
			RecordKind: p.RecordKind, AddressCidr: p.AddressCidr, OrgName: p.OrgName,
			LookupQuery: l.Query, LookupAt: l.CreatedAt, LookupBy: f.accounts[l.CreatedBy].Username,
		})
	}
	// Mirror the SQL ordering: newest lookup first, oldest proposal first.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].LookupID != rows[j].LookupID {
			return rows[i].LookupID > rows[j].LookupID
		}
		return rows[i].ID < rows[j].ID
	})
	return rows, nil
}

func (f *fakeStore) GetPendingProposal(_ context.Context, id int64) (db.Proposal, error) {
	for _, p := range f.proposals {
		if p.ID == id && p.Status == "pending" {
			return p, nil
		}
	}
	return db.Proposal{}, pgx.ErrNoRows
}

func (f *fakeStore) ConfirmProposal(_ context.Context, arg db.ConfirmProposalParams) (int64, error) {
	for i, p := range f.proposals {
		if p.ID == arg.ID && p.Status == "pending" {
			f.proposals[i].Status = "confirmed"
			f.proposals[i].ConfirmedSeedID = arg.ConfirmedSeedID
			return 1, nil
		}
	}
	return 0, nil
}

func (f *fakeStore) DeclineLookup(_ context.Context, lookupID int64) (int64, error) {
	var n int64
	for i, p := range f.proposals {
		if p.LookupID == lookupID && p.Status == "pending" {
			f.proposals[i].Status = "declined"
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) InsertReportSchedule(_ context.Context, arg db.InsertReportScheduleParams) (db.ReportSchedule, error) {
	if f.rsNextID == 0 {
		f.rsNextID = 1
	}
	rs := db.ReportSchedule{
		ID: f.rsNextID, Name: arg.Name, Sections: arg.Sections,
		Cadence: arg.Cadence, Format: arg.Format,
		DeliveryTarget: arg.DeliveryTarget, CreatedBy: arg.CreatedBy,
	}
	f.rsNextID++
	f.reportSchedules = append(f.reportSchedules, rs)
	return rs, nil
}

func (f *fakeStore) ListReportSchedules(context.Context) ([]db.ReportSchedule, error) {
	out := make([]db.ReportSchedule, len(f.reportSchedules))
	// Newest-first, mirroring ORDER BY id DESC.
	for i, rs := range f.reportSchedules {
		out[len(f.reportSchedules)-1-i] = rs
	}
	return out, nil
}

// --- SSO providers (#293) ---------------------------------------------------

func (f *fakeStore) InsertSSOProvider(_ context.Context, arg db.InsertSSOProviderParams) (int64, error) {
	for _, p := range f.ssoProviders {
		if p.slug == arg.Slug {
			return 0, &pgconn.PgError{Code: "23505", Message: "duplicate sso slug"}
		}
	}
	f.ssoNextID++
	f.ssoProviders = append(f.ssoProviders, fakeSSOProvider{
		id: f.ssoNextID, slug: arg.Slug, name: arg.Name, issuer: arg.Issuer,
		clientID: arg.ClientID, secret: arg.ClientSecret.String, hasSecret: arg.ClientSecret.Valid,
		enabled: arg.Enabled, createdBy: arg.CreatedBy,
		createdAt: obsClock,
	})
	return f.ssoNextID, nil
}

func (f *fakeStore) ListSSOProviders(context.Context) ([]db.ListSSOProvidersRow, error) {
	out := []db.ListSSOProvidersRow{}
	// Newest-first, mirroring ORDER BY id DESC. The secret is never exposed — only
	// has_secret — exactly as the query omits it.
	for i := len(f.ssoProviders) - 1; i >= 0; i-- {
		p := f.ssoProviders[i]
		out = append(out, db.ListSSOProvidersRow{
			ID: p.id, Slug: p.slug, Name: p.name, Issuer: p.issuer, ClientID: p.clientID,
			Enabled: p.enabled, HasSecret: p.hasSecret,
			CreatedBy: p.createdBy, CreatedAt: pgtype.Timestamptz{Time: p.createdAt, Valid: true},
			CreatedByUsername: f.usernameForID(p.createdBy),
		})
	}
	return out, nil
}

func (f *fakeStore) ListEnabledSSOProviders(context.Context) ([]db.ListEnabledSSOProvidersRow, error) {
	out := []db.ListEnabledSSOProvidersRow{}
	for i := len(f.ssoProviders) - 1; i >= 0; i-- {
		if p := f.ssoProviders[i]; p.enabled {
			out = append(out, db.ListEnabledSSOProvidersRow{ID: p.id, Slug: p.slug, Name: p.name})
		}
	}
	return out, nil
}

func (f *fakeStore) GetSSOProvider(_ context.Context, id int64) (db.GetSSOProviderRow, error) {
	for _, p := range f.ssoProviders {
		if p.id == id {
			return db.GetSSOProviderRow{
				ID: p.id, Slug: p.slug, Name: p.name, Issuer: p.issuer, ClientID: p.clientID,
				Enabled: p.enabled, HasSecret: p.hasSecret,
				CreatedBy: p.createdBy, CreatedAt: pgtype.Timestamptz{Time: p.createdAt, Valid: true},
			}, nil
		}
	}
	return db.GetSSOProviderRow{}, pgx.ErrNoRows
}

func (f *fakeStore) GetSSOProviderForAuth(_ context.Context, slug string) (db.GetSSOProviderForAuthRow, error) {
	for _, p := range f.ssoProviders {
		if p.slug == slug && p.enabled {
			return db.GetSSOProviderForAuthRow{
				ID: p.id, Slug: p.slug, Name: p.name, Issuer: p.issuer, ClientID: p.clientID,
				ClientSecret: pgtype.Text{String: p.secret, Valid: p.hasSecret},
			}, nil
		}
	}
	return db.GetSSOProviderForAuthRow{}, pgx.ErrNoRows
}

func (f *fakeStore) UpdateSSOProvider(_ context.Context, arg db.UpdateSSOProviderParams) (int64, error) {
	for i := range f.ssoProviders {
		if f.ssoProviders[i].id != arg.ID {
			continue
		}
		for _, p := range f.ssoProviders {
			if p.id != arg.ID && p.slug == arg.Slug {
				return 0, &pgconn.PgError{Code: "23505", Message: "duplicate sso slug"}
			}
		}
		f.ssoProviders[i].slug = arg.Slug
		f.ssoProviders[i].name = arg.Name
		f.ssoProviders[i].issuer = arg.Issuer
		f.ssoProviders[i].clientID = arg.ClientID
		f.ssoProviders[i].enabled = arg.Enabled
		return 1, nil
	}
	return 0, nil // no such id: zero rows affected, mirroring the real UPDATE
}

func (f *fakeStore) SetSSOProviderSecret(_ context.Context, arg db.SetSSOProviderSecretParams) error {
	for i := range f.ssoProviders {
		if f.ssoProviders[i].id == arg.ID {
			f.ssoProviders[i].secret = arg.ClientSecret.String
			f.ssoProviders[i].hasSecret = arg.ClientSecret.Valid
			return nil
		}
	}
	return nil
}

func (f *fakeStore) DeleteSSOProvider(_ context.Context, id int64) error {
	kept := f.ssoProviders[:0]
	for _, p := range f.ssoProviders {
		if p.id != id {
			kept = append(kept, p)
		}
	}
	f.ssoProviders = kept
	// ON DELETE CASCADE: a provider's bindings go with it.
	var keptIdents []fakeSSOIdentity
	for _, i := range f.ssoIdentities {
		if i.providerID != id {
			keptIdents = append(keptIdents, i)
		}
	}
	f.ssoIdentities = keptIdents
	return nil
}

func (f *fakeStore) InsertSSOIdentity(_ context.Context, arg db.InsertSSOIdentityParams) error {
	for _, i := range f.ssoIdentities {
		// UNIQUE(provider_id, sub): one external identity binds to one account.
		if i.providerID == arg.ProviderID && i.sub == arg.Sub {
			return &pgconn.PgError{Code: "23505", Message: "duplicate sso identity"}
		}
		// UNIQUE(provider_id, account_id): one identity per provider per account.
		if i.providerID == arg.ProviderID && i.accountID == arg.AccountID {
			return &pgconn.PgError{Code: "23505", Message: "duplicate provider link for account"}
		}
	}
	f.ssoIdentNextID++
	f.ssoIdentities = append(f.ssoIdentities, fakeSSOIdentity{
		id: f.ssoIdentNextID, providerID: arg.ProviderID, accountID: arg.AccountID,
		sub: arg.Sub, displayName: arg.DisplayName, createdAt: obsClock,
	})
	return nil
}

func (f *fakeStore) GetAccountBySSOIdentity(_ context.Context, arg db.GetAccountBySSOIdentityParams) (db.Account, error) {
	for _, i := range f.ssoIdentities {
		if i.providerID == arg.ProviderID && i.sub == arg.Sub {
			if a, ok := f.accounts[i.accountID]; ok {
				return a, nil
			}
		}
	}
	return db.Account{}, pgx.ErrNoRows
}

func (f *fakeStore) GetSSOIdentityBySub(_ context.Context, arg db.GetSSOIdentityBySubParams) (db.GetSSOIdentityBySubRow, error) {
	for _, i := range f.ssoIdentities {
		if i.providerID == arg.ProviderID && i.sub == arg.Sub {
			return db.GetSSOIdentityBySubRow{ID: i.id, AccountID: i.accountID, DisplayName: i.displayName}, nil
		}
	}
	return db.GetSSOIdentityBySubRow{}, pgx.ErrNoRows
}

func (f *fakeStore) ListSSOIdentitiesForAccount(_ context.Context, accountID int64) ([]db.ListSSOIdentitiesForAccountRow, error) {
	out := []db.ListSSOIdentitiesForAccountRow{}
	// Newest-first, mirroring ORDER BY i.id DESC.
	for k := len(f.ssoIdentities) - 1; k >= 0; k-- {
		i := f.ssoIdentities[k]
		if i.accountID != accountID {
			continue
		}
		out = append(out, db.ListSSOIdentitiesForAccountRow{
			ID: i.id, ProviderID: i.providerID,
			ProviderSlug: f.ssoSlugForID(i.providerID), ProviderName: f.ssoNameForID(i.providerID),
			DisplayName: i.displayName, CreatedAt: pgtype.Timestamptz{Time: i.createdAt, Valid: true},
		})
	}
	return out, nil
}

func (f *fakeStore) DeleteSSOIdentityForAccount(_ context.Context, arg db.DeleteSSOIdentityForAccountParams) (int64, error) {
	var kept []fakeSSOIdentity
	var removed int64
	for _, i := range f.ssoIdentities {
		if i.id == arg.ID && i.accountID == arg.AccountID {
			removed++
			continue
		}
		kept = append(kept, i)
	}
	f.ssoIdentities = kept
	return removed, nil
}

func (f *fakeStore) ListSSOBindings(_ context.Context) ([]db.ListSSOBindingsRow, error) {
	out := []db.ListSSOBindingsRow{}
	for k := len(f.ssoIdentities) - 1; k >= 0; k-- {
		i := f.ssoIdentities[k]
		out = append(out, db.ListSSOBindingsRow{
			ID: i.id, ProviderID: i.providerID,
			ProviderSlug: f.ssoSlugForID(i.providerID), ProviderName: f.ssoNameForID(i.providerID),
			AccountID: i.accountID, AccountUsername: f.usernameForID(i.accountID),
			DisplayName: i.displayName, CreatedAt: pgtype.Timestamptz{Time: i.createdAt, Valid: true},
		})
	}
	return out, nil
}

func (f *fakeStore) DeleteSSOIdentity(_ context.Context, id int64) error {
	var kept []fakeSSOIdentity
	for _, i := range f.ssoIdentities {
		if i.id != id {
			kept = append(kept, i)
		}
	}
	f.ssoIdentities = kept
	return nil
}

// ssoSlugForID / ssoNameForID resolve a provider id to its slug/name for the identity
// join queries; an unknown id renders empty.
func (f *fakeStore) ssoSlugForID(id int64) string {
	for _, p := range f.ssoProviders {
		if p.id == id {
			return p.slug
		}
	}
	return ""
}

func (f *fakeStore) ssoNameForID(id int64) string {
	for _, p := range f.ssoProviders {
		if p.id == id {
			return p.name
		}
	}
	return ""
}

// usernameForID resolves an account id to its username for the created-by join the
// SSO list query performs; an unknown id renders empty (a test rarely asserts it).
func (f *fakeStore) usernameForID(id int64) string {
	return f.accounts[id].Username
}

// testKey is a fixed 32-byte session signing key for tests.
var testKey = []byte("0123456789abcdef0123456789abcdef")

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC) }
}

func TestHealthzOK(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	f.hb = db.Heartbeat{ID: 1, CheckedAt: pgtype.Timestamptz{Time: now, Valid: true}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newServer(f, testKey, "", fixedClock()).handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Status    string    `json:"status"`
		CheckedAt time.Time `json:"checked_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" || !body.CheckedAt.Equal(now) {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestHealthzDBError(t *testing.T) {
	f := newFakeStore()
	f.hbErr = errors.New("connection refused")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newServer(f, testKey, "", fixedClock()).handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

// TestDeprecatedRoutesReconciled is the #286 IA-reconciliation proof: each retired
// GET answers the right redirect to its canonical home, while every viewer-readable
// fold keeps resolving for a viewer — no 404, and no viewer bounced into an
// admin-gated 403 (#281 caveat). The detail deep-links under /subjects/* are NOT
// redirected: Inventory rows link straight to them and they are the detail pages.
func TestDeprecatedRoutesReconciled(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

	// Retired GETs redirect to their canonical home (301 permanent for pure moves).
	for _, tc := range []struct {
		path, want string
	}{
		{"/seeds", "/scope"},
		{"/subjects", "/inventory"},
	} {
		resp, err := vc.Get(base + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != tc.want {
			t.Errorf("GET %s: status=%d location=%q, want 301 -> %s",
				tc.path, resp.StatusCode, resp.Header.Get("Location"), tc.want)
		}
	}

	// The viewer-readable folds keep resolving for a viewer — never redirected into
	// admin-gated Settings, never a 403. /coverage stays as the distinct aperture
	// artifact; /messages stays viewer-readable as the messages fold (the V3 shell
	// bell now targets /inbox, T4) for all users. /exposure is now the first-class
	// Exposure page (#300, repurposed from its #286 redirect); a viewer reads it (its
	// WITHHELD/board states are covered in exposure_test.go).
	for _, path := range []string{"/messages", "/scans", "/verge-core", "/sources", "/coverage", "/exposure"} {
		resp, err := vc.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET %s (viewer): status=%d, want 200 (kept viewer-readable)", path, resp.StatusCode)
		}
	}

	// The subject detail deep-links are preserved (not redirected): a missing key
	// renders the 404 detail page, proving the routes still resolve to the handler.
	for _, path := range []string{"/subjects/never.measured.example", "/subjects/service?key=x%3A1%2Ftcp"} {
		resp, err := vc.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status=%d, want 404 (detail route still resolves)", path, resp.StatusCode)
		}
	}
}
