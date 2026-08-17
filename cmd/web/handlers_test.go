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
	msgNextID     int64
	previewResult db.PreviewExclusionWithdrawalRow

	// dispatchProgress and jobsByDispatch stand in for the queue reads behind the
	// Scans monitor (#245); the scans test seeds them directly.
	dispatchProgress []db.ListDispatchProgressRow
	jobsByDispatch   map[int64][]db.ListJobsForDispatchRow
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
		sourceStates: map[string]db.SourceState{},
		scans: []db.Scan{
			{ID: 1, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
			{ID: 2, Kind: "hot", Enabled: true, CadenceSeconds: 86400},
			// The cold Scan ships disabled with an empty scope list (ADR-0044).
			{ID: 3, Kind: "cold", Enabled: false, CadenceSeconds: 2592000},
		},
		obsNextID: 1, batchNextID: 1, scanNextID: 1,
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

func (f *fakeStore) CountUnreadMessages(context.Context) (int64, error) {
	var n int64
	for _, m := range f.messages {
		if !m.ReadAt.Valid {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) MarkMessageRead(_ context.Context, arg db.MarkMessageReadParams) error {
	for i := range f.messages {
		if f.messages[i].ID == arg.ID && !f.messages[i].ReadAt.Valid {
			f.messages[i].ReadAt = arg.ReadAt
		}
	}
	return nil
}

func (f *fakeStore) MarkAllMessagesRead(_ context.Context, readAt pgtype.Timestamptz) error {
	for i := range f.messages {
		if !f.messages[i].ReadAt.Valid {
			f.messages[i].ReadAt = readAt
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
	b := f.freshBatch("ct", "ct")
	f.admitted = append(f.admitted, db.AdmittedName{
		ID: int64(len(f.admitted) + 1), Name: name, Source: "crtsh", BatchID: b,
		CreatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
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
