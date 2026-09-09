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
	"sync"
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

// A posed read derives nothing, so a fixture supplies rows already past the query's filters.

type fakeStore struct {
	hb    db.Heartbeat
	hbErr error

	// The map detector trips on a concurrent TOTP login a live Postgres would serialise.

	acctMu   sync.Mutex
	accounts map[int64]db.Account
	byName   map[string]int64
	nextID   int64

	seeds      []db.Seed
	seedNextID int64

	seedWithdrawals []db.SeedWithdrawal

	withdrawalCandidates []db.ListSeedWithdrawalCandidatesRow

	nameWithdrawalCandidates []db.ListNameSeedWithdrawalCandidatesRow

	cited               []db.NameCitedAddressesRow
	citedErr            error
	edgeFanout          []db.ListEdgeFanoutMeasurementsRow
	certMaterial        map[string][]byte
	completedBatchKinds map[string]bool

	edgeFanoutBounds [][]string

	exclusions    []db.Exclusion
	exclNextID    int64
	exclusionsErr error

	annotations []db.Annotation
	annoNextID  int64

	signalInstances  []db.SignalInstance
	signalInstNextID int64

	withdrawalLifespans []db.ListWithdrawalLifespansRow

	subjectClosures map[fakeSubjectRef]time.Time

	sourceStates map[string]db.SourceState

	sourceHealth map[string]db.SourceHealth

	ctReliability map[string]db.CTReliabilityWindowRow

	ctAdmitCount int64

	ctTailBatch db.CTTailLastBatchRow

	certMaterialCount int64

	integrationStates map[string]db.IntegrationState

	personalTokens []db.PersonalToken
	tokenNextID    int64

	sessions      []db.Session
	sessionNextID int64

	passwordResets  []db.PasswordReset
	resetNextID     int64
	recoveryCodes   []db.RecoveryCode
	recoveryNextID  int64
	invites         []db.Invite
	inviteStaleRead bool
	inviteNextID    int64

	admitted []db.AdmittedName

	vantages      []db.Vantage
	vantageNextID int64
	vantagesErr   error

	driftEventsErr  error
	reachSpansErr   error
	pendingPropsErr error

	channels       []fakeChannel
	chanNextID     int64
	retention      db.GetRetentionSettingsRow
	retentionErr   error
	facetFloors    []db.ListFacetSourceFloorsRow
	derivBreaks    []db.ListDerivationBreaksRow
	heldObs        int64
	heldEstimate   int64
	instanceConfig db.GetInstanceConfigRow

	observations    []db.Observation
	batches         []db.Batch
	scans           []db.Scan
	listScansErr    error
	coveringScans   []string
	coveringScanErr error
	undoDeclineErr  error
	obsNextID       int64
	batchNextID     int64
	scanNextID      int64

	zoneFiles    []fakeZoneFile
	zoneNextID   int64
	zoneCadence  int64
	dnsCadence   int64
	lookups      []db.ProposerLookup
	lookupNextID int64
	proposals    []db.Proposal
	proposalNext int64
	declinedRead int

	freqEdits map[int32]fakeFreqEdit

	coldScopes map[int64]bool

	messages         []db.Message
	deliveryOutcomes []db.ListDeliveryOutcomesRow
	messageRead      map[int64]map[int64]bool
	msgNextID        int64
	previewResult    db.PreviewExclusionWithdrawalRow

	dispatchProgress []db.ListDispatchProgressRow
	jobsByDispatch   map[int64][]db.ListJobsForDispatchRow
	transcriptsByJob map[int64]db.Transcript
	dispatchStatus   map[int64]string
	instanceHealth   db.GetInstanceHealthRow

	reportSchedules []db.ReportSchedule
	rsNextID        int64

	reportDeliveries []db.ReportDelivery
	rdNextID         int64

	ssoProviders []fakeSSOProvider
	ssoNextID    int64

	ssoIdentities  []fakeSSOIdentity
	ssoIdentNextID int64
}

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

type fakeSSOIdentity struct {
	id          int64
	providerID  int64
	accountID   int64
	sub         string
	displayName string
	createdAt   time.Time
}

type fakeFreqEdit struct {
	action    string
	createdBy int64
}

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
		lookupNextID: 1, proposalNext: 1, signalInstNextID: 1000,
		sourceStates:      map[string]db.SourceState{},
		sourceHealth:      map[string]db.SourceHealth{},
		integrationStates: map[string]db.IntegrationState{},
		scans: []db.Scan{
			{ID: 1, Kind: "dns", Enabled: true, CadenceSeconds: 86400},
			{ID: 2, Kind: "hot", Enabled: true, CadenceSeconds: 86400},
			{ID: 3, Kind: "cold", Enabled: false, CadenceSeconds: 2592000},
		},
		obsNextID: 1, batchNextID: 1, scanNextID: 1, tokenNextID: 1,
		resetNextID: 1, recoveryNextID: 1, inviteNextID: 1, sessionNextID: 1,
		freqEdits:           map[int32]fakeFreqEdit{},
		coldScopes:          map[int64]bool{},
		completedBatchKinds: map[string]bool{},
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

func (f *fakeStore) dispatchIdx(id int64) int {
	for i := range f.dispatchProgress {
		if f.dispatchProgress[i].DispatchID == id {
			return i
		}
	}
	return -1
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

func (f *fakeStore) RecordHeartbeat(context.Context) (db.Heartbeat, error) {
	return f.hb, f.hbErr
}

func (f *fakeStore) GetAccountByUsername(_ context.Context, username string) (db.Account, error) {
	f.acctMu.Lock()
	defer f.acctMu.Unlock()
	id, ok := f.byName[username]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return f.accounts[id], nil
}

func (f *fakeStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	f.acctMu.Lock()
	defer f.acctMu.Unlock()
	acct, ok := f.accounts[id]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return acct, nil
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

func (f *fakeStore) ListAddressScopeCidrs(context.Context) ([]*netip.Prefix, error) {
	out := []*netip.Prefix{}
	for _, s := range f.seeds {
		if s.Kind == "address" && s.AddressCidr != nil {
			out = append(out, s.AddressCidr)
		}
	}
	// The fake returns this scope even where no seed declares it, so class fixtures need none.
	conv := netip.MustParsePrefix("10.0.0.0/8")
	out = append(out, &conv)
	return out, nil
}

func (f *fakeStore) ListExtendedZoneDomains(context.Context) ([]pgtype.Text, error) {
	out := []pgtype.Text{}
	for _, s := range f.seeds {
		if s.Kind == "name" && s.CustodyExtension && s.NameDomain.Valid {
			out = append(out, s.NameDomain)
		}
	}
	return out, nil
}

func (f *fakeStore) NameCitedAddresses(context.Context, db.NameCitedAddressesParams) ([]db.NameCitedAddressesRow, error) {
	if f.citedErr != nil {
		return nil, f.citedErr
	}
	return f.cited, nil
}

func (f *fakeStore) ListEdgeFanoutMeasurements(context.Context) ([]db.ListEdgeFanoutMeasurementsRow, error) {
	return f.edgeFanout, nil
}

func (f *fakeStore) ListEdgeFanoutMeasurementsOver(_ context.Context, addresses []string) ([]db.ListEdgeFanoutMeasurementsOverRow, error) {
	f.edgeFanoutBounds = append(f.edgeFanoutBounds, addresses)
	want := make(map[string]struct{}, len(addresses))
	for _, a := range addresses {
		want[a] = struct{}{}
	}
	out := []db.ListEdgeFanoutMeasurementsOverRow{}
	for _, r := range f.edgeFanout {
		if _, asked := want[r.Address]; !asked {
			continue
		}
		out = append(out, db.ListEdgeFanoutMeasurementsOverRow{
			Address: r.Address, Outcome: r.Outcome, Fingerprint: r.Fingerprint,
		})
	}
	return out, nil
}

func (f *fakeStore) ListCertificateMaterialDER(_ context.Context, fingerprints []string) ([]db.ListCertificateMaterialDERRow, error) {
	out := []db.ListCertificateMaterialDERRow{}
	for _, fp := range fingerprints {
		if der, held := f.certMaterial[fp]; held {
			out = append(out, db.ListCertificateMaterialDERRow{Fingerprint: fp, Der: der})
		}
	}
	return out, nil
}

func (f *fakeStore) measuredEdge(addr, outcome string, der []byte) {
	row := db.ListEdgeFanoutMeasurementsRow{Address: addr, Outcome: outcome}
	if len(der) > 0 {
		fingerprint := connectoutcome.Fingerprint(der)
		row.Fingerprint = pgtype.Text{String: fingerprint, Valid: true}
		if f.certMaterial == nil {
			f.certMaterial = map[string][]byte{}
		}
		f.certMaterial[fingerprint] = der
	}
	f.edgeFanout = append(f.edgeFanout, row)
}

func (f *fakeStore) ScanHasCompletedBatch(_ context.Context, kind string) (bool, error) {
	return f.completedBatchKinds[kind], nil
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

func (f *fakeStore) ListAddressExclusionCidrs(context.Context) ([]*netip.Prefix, error) {
	out := []*netip.Prefix{}
	for _, e := range f.exclusions {
		if e.Kind == "address" && e.AddressCidr != nil {
			out = append(out, e.AddressCidr)
		}
	}
	return out, nil
}

func (f *fakeStore) ListVantages(context.Context) ([]db.ListVantagesRow, error) {
	if f.vantagesErr != nil {
		return nil, f.vantagesErr
	}
	rows := make([]db.ListVantagesRow, 0, len(f.vantages))
	for i := len(f.vantages) - 1; i >= 0; i-- {
		v := f.vantages[i]
		if !v.Host.Valid {
			continue
		}
		rows = append(rows, db.ListVantagesRow{
			ID: v.ID, Name: v.Name, Class: v.Class, Resolver: v.Resolver,
			Host: v.Host, Port: v.Port, Username: v.Username,
			Availability: v.Availability, PublicKey: v.PublicKey, HostKey: v.HostKey,
			CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, LatencyMs: v.LatencyMs,
			Platform: v.Platform, Egress: v.Egress, DialledAddr: v.DialledAddr,
			CreatedByUsername: f.accounts[v.CreatedBy.Int64].Username,
			Observed:          f.vantageObserved(v.ID),
		})
	}
	return rows, nil
}

func (f *fakeStore) ListUnavailableVantages(context.Context) ([]db.ListUnavailableVantagesRow, error) {
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

func (f *fakeStore) ListSignalInstances(context.Context) ([]db.SignalInstance, error) {
	rows := append([]db.SignalInstance(nil), f.signalInstances...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SignalName != rows[j].SignalName {
			return rows[i].SignalName < rows[j].SignalName
		}
		return rows[i].SubjectKey < rows[j].SubjectKey
	})
	return rows, nil
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

func (f *fakeStore) ListAccounts(context.Context) ([]db.ListAccountsRow, error) {
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

func (f *fakeStore) ListChannels(context.Context) ([]db.ListChannelsRow, error) {
	rows := make([]db.ListChannelsRow, 0, len(f.channels))
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

func (f *fakeStore) GetInstanceConfig(context.Context) (db.GetInstanceConfigRow, error) {
	return f.instanceConfig, nil
}

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

func (f *fakeStore) freshBatch(scanKind, batchKind string) int64 {
	// An observation whose batch cites no enabled Scan has no bound and is never live (#237).
	scanID := f.ensureScan(scanKind)
	b := db.Batch{ID: f.batchNextID, ScanID: scanID, Kind: batchKind, Outcome: "completed"}
	f.batches = append(f.batches, b)
	f.batchNextID++
	return b.ID
}

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

func (f *fakeStore) addAdmittedName(t *testing.T, name string, at time.Time) {
	t.Helper()
	f.addAdmittedNameUnderSeed(t, name, f.coveringNameSeedID(name), at)
}

func (f *fakeStore) addAdmittedNameUnderSeed(t *testing.T, name string, seedID int64, at time.Time) {
	t.Helper()
	b := f.freshBatch("ct", "ct")
	f.admitted = append(f.admitted, db.AdmittedName{
		ID: int64(len(f.admitted) + 1), Name: name, Source: "crtsh", SeedID: seedID, BatchID: b,
		CreatedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
}

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

func (f *fakeStore) liveObservations(asOf time.Time) []db.Observation {
	enabledCadence := map[int64]int64{}
	for _, sc := range f.scans {
		if sc.Enabled {
			enabledCadence[sc.ID] = sc.CadenceSeconds
		}
	}
	batchCadence := map[int64]int64{}
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
	tightest := map[timeline]int64{}
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

func (f *fakeStore) ListZoneFileStatus(context.Context) ([]db.ListZoneFileStatusRow, error) {
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
		for _, s := range f.foldWithClosure(key, byKey[k]) {
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

type fakeSubjectRef struct{ kind, key string }

func (f *fakeStore) withdrawSubject(kind, key string, at time.Time) {
	if f.subjectClosures == nil {
		f.subjectClosures = map[fakeSubjectRef]time.Time{}
	}
	f.subjectClosures[fakeSubjectRef{kind: kind, key: key}] = at
}

func (f *fakeStore) foldWithClosure(key drift.TimelineKey, readings []drift.Reading) []drift.Span {
	spans := drift.Fold(key, readings)
	if at, ok := f.subjectClosures[fakeSubjectRef{kind: key.SubjectKind, key: key.SubjectKey}]; ok {
		spans = drift.CloseWithdrawal(spans, at, drift.ReasonMeasuredAbsent)
	}
	return spans
}

func (f *fakeStore) fakeBatchByID(id int64) db.Batch {
	for _, b := range f.batches {
		if b.ID == id {
			return b
		}
	}
	return db.Batch{}
}

func (f *fakeStore) ListRecentDriftEvents(_ context.Context, arg db.ListRecentDriftEventsParams) ([]db.ListRecentDriftEventsRow, error) {
	if f.driftEventsErr != nil {
		return nil, f.driftEventsErr
	}
	since := arg.Since
	type tlkey struct{ kind, key, facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	obsBatch := map[tlkey]map[int64]int64{}
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
			if arg.Until.Valid && !s.OpenedAt.Before(arg.Until.Time) {
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
	if arg.MaxEvents > 0 && int32(len(rows)) > arg.MaxEvents {
		rows = rows[:arg.MaxEvents]
	}
	return rows, nil
}

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

type reachVantageKey struct {
	svc     string
	vantage int64
}

func (f *fakeStore) currentReachByVantage() map[reachVantageKey]db.Observation {
	// The span corpus has no retention policy, so this read applies no live-tier gate (ADR-0041).
	known := map[int64]bool{}
	for _, v := range f.vantages {
		known[v.ID] = true
	}
	latest := map[reachVantageKey]db.Observation{}
	for _, o := range f.observations {
		if o.SubjectKind != "service" || o.Facet != "reachability" || !o.VantageID.Valid {
			continue
		}
		if !known[o.VantageID.Int64] {
			continue
		}
		k := reachVantageKey{o.SubjectKey, o.VantageID.Int64}
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
	if f.reachSpansErr != nil {
		return nil, f.reachSpansErr
	}
	rows := []db.ListServiceReachabilitySpansByClassRow{}
	for k, o := range f.currentReachByVantage() {
		v := f.vantageByID(k.vantage)
		rows = append(rows, db.ListServiceReachabilitySpansByClassRow{
			SubjectKey: k.svc, VantageID: pgtype.Int8{Int64: k.vantage, Valid: true},
			Value: o.Value, IsGap: reachOutcomeIsGap(o.Value),
			OpenedAt: o.ObservedAt, ID: o.ID,
			Host: v.Host, Egress: v.Egress, DialledAddr: v.DialledAddr,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].SubjectKey != rows[j].SubjectKey {
			return rows[i].SubjectKey < rows[j].SubjectKey
		}
		return rows[i].VantageID.Int64 < rows[j].VantageID.Int64
	})
	return rows, nil
}

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
		rows = append(rows, db.ListEndpointCertificatesRow{SubjectKey: k, Value: o.Value, ObservedAt: o.ObservedAt})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].SubjectKey < rows[j].SubjectKey })
	return rows, nil
}

func (f *fakeStore) addTLSAcceptance(t *testing.T, serviceKey string, at time.Time, value string) {
	t.Helper()
	b := f.freshBatch("tls-acceptance", "tls-acceptance")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "tls-acceptance", SubjectKind: "service",
		SubjectKey: serviceKey, Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
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

func (f *fakeStore) addVantageClass(class string) int64 {
	v := db.Vantage{
		ID: f.vantageNextID, Name: class + "-resolver", Class: class,
		DialledAddr: classPresentedDialled(class),
	}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v.ID
}

func classPresentedDialled(class string) pgtype.Text {
	switch class {
	case "internet":
		return pgtype.Text{String: "198.51.100.200", Valid: true}
	case "internal":
		return pgtype.Text{String: "10.200.0.1", Valid: true}
	default:
		return pgtype.Text{}
	}
}

func (f *fakeStore) vantageByID(id int64) db.Vantage {
	for _, v := range f.vantages {
		if v.ID == id {
			return v
		}
	}
	return db.Vantage{}
}

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

func (f *fakeStore) GetZoneCadenceSeconds(context.Context) (int64, error) {
	if f.zoneCadence == 0 {
		return 2592000, nil
	}
	return f.zoneCadence, nil
}

func (f *fakeStore) GetDnsCadenceSeconds(context.Context) (int64, error) {
	if f.dnsCadence == 0 {
		return 86400, nil
	}
	return f.dnsCadence, nil
}

func (f *fakeStore) ListReportSchedules(context.Context) ([]db.ReportSchedule, error) {
	out := make([]db.ReportSchedule, len(f.reportSchedules))
	for i, rs := range f.reportSchedules {
		out[len(f.reportSchedules)-1-i] = rs
	}
	return out, nil
}

func (f *fakeStore) GetLatestReportDelivery(_ context.Context, scheduleID int64) (db.ReportDelivery, error) {
	var latest db.ReportDelivery
	found := false
	for _, d := range f.reportDeliveries {
		if d.ScheduleID != scheduleID || d.State == "failed" {
			continue
		}
		if !found || d.ID > latest.ID {
			latest, found = d, true
		}
	}
	if !found {
		return db.ReportDelivery{}, pgx.ErrNoRows
	}
	return latest, nil
}

func (f *fakeStore) ListReportDeliveries(_ context.Context, scheduleID int64) ([]db.ReportDelivery, error) {
	out := []db.ReportDelivery{}
	for i := len(f.reportDeliveries) - 1; i >= 0; i-- {
		if f.reportDeliveries[i].ScheduleID == scheduleID {
			out = append(out, f.reportDeliveries[i])
		}
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

func (f *fakeStore) usernameForID(id int64) string {
	return f.accounts[id].Username
}

var testKey = []byte("0123456789abcdef0123456789abcdef")

var testTranscriptKey = []byte("transcriptkey0123456789abcdef012")

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

func TestDeprecatedRoutesReconciled(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "viewer", roleViewer, "hunter2hunter2")
	base := start(t, f, "")
	vc := login(t, base, "viewer", "hunter2hunter2")

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
