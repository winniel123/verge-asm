package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
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

	sourceStates map[string]db.SourceState

	vantages      []db.Vantage
	vantageNextID int64

	channels   []fakeChannel
	chanNextID int64
	retention  db.GetRetentionSettingsRow

	// scans mirrors the scan table. newFakeStore seeds the dns Scan the migration
	// ships (enabled, daily) so the aperture statement has a cadence to read.
	// The Subjects reads (#189) also join the observation/batch corpus.
	observations []db.Observation
	batches      []db.Batch
	scans        []db.Scan
	obsNextID    int64
	batchNextID  int64
	scanNextID   int64

	zoneFiles   []fakeZoneFile
	zoneNextID  int64
	zoneCadence int64
	lookups      []db.ProposerLookup
	lookupNextID int64
	proposals    []db.Proposal
	proposalNext int64
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
		seedNextID: 1, exclNextID: 1, vantageNextID: 1, chanNextID: 1,
		lookupNextID: 1, proposalNext: 1,
		sourceStates: map[string]db.SourceState{},
		scans: []db.Scan{
			{ID: 1, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
		},
		obsNextID: 1, batchNextID: 1, scanNextID: 1,
	}
}

func (f *fakeStore) GetScanByKind(_ context.Context, kind string) (db.Scan, error) {
	for _, sc := range f.scans {
		if sc.Kind == kind {
			return sc, nil
		}
	}
	return db.Scan{}, pgx.ErrNoRows
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

func (f *fakeStore) DeleteExclusion(_ context.Context, id int64) error {
	for i, e := range f.exclusions {
		if e.ID == id {
			f.exclusions = append(f.exclusions[:i], f.exclusions[i+1:]...)
			return nil
		}
	}
	return nil // idempotent: a missing row is not an error
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

// addResolution records a resolution observation for a Name in a fresh batch of
// the given scan kind, mirroring what the measurement worker writes. It is the
// only seam the Subjects tests need to populate the estate.
func (f *fakeStore) addResolution(t *testing.T, createdBy int64, name, scanKind string, at time.Time, value string) {
	t.Helper()
	scanID := f.ensureScan(scanKind)
	b := db.Batch{ID: f.batchNextID, ScanID: scanID, Kind: "resolution-walk", Outcome: "completed"}
	f.batches = append(f.batches, b)
	f.batchNextID++
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b.ID, Facet: "resolution", SubjectKind: "name",
		SubjectKey: name, Source: "resolver", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

// latestResolutionByName picks, per Name, its latest resolution observation —
// max observed_at, then max id — mirroring the DISTINCT ON in the SQL.
func (f *fakeStore) latestResolutionByName() map[string]db.Observation {
	latest := map[string]db.Observation{}
	for _, o := range f.observations {
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

func (f *fakeStore) ListCurrentNameSubjects(_ context.Context, search string) ([]db.ListCurrentNameSubjectsRow, error) {
	latest := f.latestResolutionByName()
	keys := make([]string, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := []db.ListCurrentNameSubjectsRow{}
	for _, k := range keys {
		o := latest[k]
		if fakeResolutionOutcome(o.Value) == "NameError" {
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

// ListSpansForSubject folds the fake's observations for one subject into Span
// rows using the real drift.Fold, so the drill-down test exercises the same
// open/close logic the worker's ingest does. The production store reads persisted
// spans; the fake derives them so the web tests stay hermetic.
func (f *fakeStore) ListSpansForSubject(_ context.Context, arg db.ListSpansForSubjectParams) ([]db.ListSpansForSubjectRow, error) {
	// Both resolution and dns-record are decided by the two membership leaves
	// jointly (ADR-0086), so every fold carries the same two-leaf vector.
	vector := drift.NewVector(
		drift.Component{Leaf: "resolution-walk", Version: resolutionwalk.Version},
		drift.Component{Leaf: "wildcard-discrimination", Version: wildcarddiscrim.Version},
	)
	derivation, _ := json.Marshal(vector)

	type tlkey struct{ facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	for _, o := range f.observations {
		if o.SubjectKind != "name" || o.SubjectKey != arg.SubjectKey {
			continue
		}
		k := tlkey{facet: o.Facet, discriminator: o.Discriminator, source: o.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		gap := o.Facet == "resolution" && fakeResolutionOutcome(o.Value) == "Gap"
		byKey[k] = append(byKey[k], drift.Reading{
			Value: string(o.Value), IsGap: gap, Vector: vector, ObservedAt: o.ObservedAt.Time,
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
		key := drift.TimelineKey{
			SubjectKind: "name", SubjectKey: arg.SubjectKey,
			Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
		}
		for _, s := range drift.Fold(key, byKey[k]) {
			id++
			row := db.ListSpansForSubjectRow{
				ID: id, SubjectKind: "name", SubjectKey: arg.SubjectKey,
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

func (f *fakeStore) GetNameSubject(_ context.Context, key string) (db.GetNameSubjectRow, error) {
	o, ok := f.latestResolutionByName()[key]
	if !ok {
		return db.GetNameSubjectRow{}, pgx.ErrNoRows
	}
	return db.GetNameSubjectRow{SubjectKey: key, Value: o.Value, ObservedAt: o.ObservedAt}, nil
}

func (f *fakeStore) GetNameCitation(_ context.Context, key string) (db.GetNameCitationRow, error) {
	var best *db.Observation
	for i := range f.observations {
		o := &f.observations[i]
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
	var scanID int64
	for _, b := range f.batches {
		if b.ID == best.BatchID {
			scanID = b.ScanID
		}
	}
	var scanKind string
	for _, sc := range f.scans {
		if sc.ID == scanID {
			scanKind = sc.Kind
		}
	}
	return db.GetNameCitationRow{
		ID: best.ID, ObservedAt: best.ObservedAt, Source: best.Source,
		VantageID: best.VantageID, BatchID: best.BatchID, ScanID: scanID, ScanKind: scanKind,
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
