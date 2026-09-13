package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

const (
	heldRootValue = `{"outcome":"Resolved","addresses":["` + apexAddr + `"]}`
	heldCause     = apexName + " entered the estate"
)

type fakeReleaseStore struct {
	opened []db.ListSubjectsOpenedSinceBatchRow

	asked       []db.ListSubjectsOpenedSinceBatchParams
	released    []db.ReleaseHeldMessageParams
	claimedRows int64

	openedErr error
}

func (f *fakeReleaseStore) ListSubjectsOpenedSinceBatch(_ context.Context, arg db.ListSubjectsOpenedSinceBatchParams) ([]db.ListSubjectsOpenedSinceBatchRow, error) {
	f.asked = append(f.asked, arg)
	return f.opened, f.openedErr
}

func (f *fakeReleaseStore) ReleaseHeldMessage(_ context.Context, arg db.ReleaseHeldMessageParams) (int64, error) {
	f.released = append(f.released, arg)
	return f.claimedRows, nil
}

func heldRow(t *testing.T, id, batchID int64) db.ListReleasableHeldMessagesRow {
	t.Helper()
	basis, err := message.CensusBasis{RootKind: subjectKindName, RootKey: apexName, RootValue: []byte(heldRootValue)}.Marshal()
	if err != nil {
		t.Fatalf("marshal basis: %v", err)
	}
	return db.ListReleasableHeldMessagesRow{
		ID:                      id,
		Class:                   string(message.ClassDrift),
		Headline:                heldCause,
		CensusPendingAfterBatch: pgtype.Int8{Int64: batchID, Valid: true},
		CensusBasis:             basis,
		// No drained dispatch leaves the root's own batch as both bounds (ADR-1806 §6).
		CensusUpperBatch: batchID,
	}
}

func subjectRow(kind, key string) db.ListSubjectsOpenedSinceBatchRow {
	return db.ListSubjectsOpenedSinceBatchRow{SubjectKind: kind, SubjectKey: key}
}

func TestReleaseNamesWhatOpenedBeneathTheRoot(t *testing.T) {
	// The census names what the hot tier opened beneath the Name, and the headline is the fold's
	// own cause clause plus that count (ADR-1806 §2, #1774).
	row := heldRow(t, 41, 9)
	store := &fakeReleaseStore{
		opened: []db.ListSubjectsOpenedSinceBatchRow{
			subjectRow(subjectKindService, apexSvc),
			subjectRow(subjectKindEndpoint, apexName+"@"+apexSvc),
			subjectRow(subjectKindEndpoint, subName+"@"+uncitedSvc),
			subjectRow(subjectKindName, subName),
		},
		claimedRows: 1,
	}
	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, row, fakeEnqueuer(2, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if !took {
		t.Fatal("the pass claimed the row, so it released it")
	}
	if len(store.asked) != 1 || store.asked[0].BatchID != 9 {
		t.Errorf("the read is bounded below by the root's own batch, got %+v", store.asked)
	}

	census, err := message.ParseCensus(store.released[0].Census)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	keys := map[string]bool{}
	for _, e := range census.Entries {
		keys[e.Key] = true
	}
	if !keys[apexSvc] || !keys[apexName+"@"+apexSvc] {
		t.Errorf("the cited Service and the root's own Endpoint are beneath it, got %+v", census.Entries)
	}
	if keys[subName+"@"+uncitedSvc] {
		t.Errorf("the frozen basis cites no address of that Service, got %+v", census.Entries)
	}
	if keys[subName] {
		t.Errorf("a census admits a Service or an Endpoint only, got %+v", census.Entries)
	}

	want := message.ReleasedMembershipHeadline(heldCause, subjectKindName, census)
	if got := store.released[0].Headline; got != want {
		t.Errorf("headline = %q, want the cause clause plus the census clause %q", got, want)
	}
	if len(log) != 1 || log[0].messageID != 41 || log[0].class != message.ClassDrift {
		t.Errorf("a released row enqueues its delivery once, got %+v", log)
	}
}

func TestReleaseReadsTheFrozenBasisAndNotLiveResolution(t *testing.T) {
	// The basis is frozen at the cause, so a re-point after it adds no subject (ADR-1806 §4).
	store := &fakeReleaseStore{
		opened: []db.ListSubjectsOpenedSinceBatchRow{
			subjectRow(subjectKindService, apexSvc),
			subjectRow(subjectKindService, uncitedSvc),
		},
		claimedRows: 1,
	}
	var log []routed
	if _, err := releaseHeldMessage(context.Background(), store, heldRow(t, 7, 3), fakeEnqueuer(1, &log)); err != nil {
		t.Fatalf("release: %v", err)
	}
	census, err := message.ParseCensus(store.released[0].Census)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	if census.Len() != 1 || census.Entries[0].Key != apexSvc {
		t.Errorf("only the address the basis cites is beneath the root, got %+v", census.Entries)
	}
}

func TestASecondPassOverAReleasedRowChangesNothing(t *testing.T) {
	// A second pass takes no row, so it enqueues nothing either (ADR-1806 §2).
	store := &fakeReleaseStore{
		opened:      []db.ListSubjectsOpenedSinceBatchRow{subjectRow(subjectKindService, apexSvc)},
		claimedRows: 0,
	}
	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, heldRow(t, 12, 4), fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if took {
		t.Error("the update took no row, so the pass lost the claim")
	}
	if len(log) != 0 {
		t.Errorf("a pass that took no row owes no delivery, got %+v", log)
	}
}

func TestReleaseWritesAnEmptyCensusRatherThanHolding(t *testing.T) {
	// The tier drained and opened nothing, which is the empty census a held row may not be
	// confused with (ADR-1806 §7).
	store := &fakeReleaseStore{claimedRows: 1}
	var log []routed
	took, err := releaseHeldMessage(context.Background(), store, heldRow(t, 5, 2), fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if !took {
		t.Fatal("a drained tier that opened nothing still releases the row")
	}
	census, err := message.ParseCensus(store.released[0].Census)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	if census.Len() != 0 {
		t.Errorf("census = %+v, want none", census.Entries)
	}
	if got := store.released[0].Headline; got != heldCause+" · 0 timelines opened beneath it" {
		t.Errorf("headline = %q, want the cause clause and a zero count", got)
	}
}

func TestReleaseReadsTheRootFromTheBasisAndNotTheFiredAtSubject(t *testing.T) {
	// A revealed firing fires at the Seed and names no root, so the basis is the only root there.
	row := heldRow(t, 21, 6)
	basis, err := message.CensusBasis{RootKind: subjectKindAddress, RootKey: apexAddr}.Marshal()
	if err != nil {
		t.Fatalf("marshal basis: %v", err)
	}
	row.CensusBasis = basis
	store := &fakeReleaseStore{
		opened:      []db.ListSubjectsOpenedSinceBatchRow{subjectRow(subjectKindService, apexSvc)},
		claimedRows: 1,
	}
	var log []routed
	if _, err := releaseHeldMessage(context.Background(), store, row, fakeEnqueuer(1, &log)); err != nil {
		t.Fatalf("release: %v", err)
	}
	census, err := message.ParseCensus(store.released[0].Census)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	if census.Len() != 1 || census.Entries[0].Key != apexSvc {
		t.Errorf("the Service sits on the Address root, got %+v", census.Entries)
	}
}

func TestTheReadStopsAtTheDispatchThatAnsweredTheRelease(t *testing.T) {
	// The release poll chose one drained dispatch, and the census names what that dispatch
	// opened and nothing after it. An open top folds a later cause's subjects into this
	// message under this message's instant, which is what a frozen basis prevents down the
	// other axis (ADR-1806 §4, ADR-1870 §2).
	row := heldRow(t, 31, 9)
	row.CensusUpperBatch = 14
	store := &fakeReleaseStore{claimedRows: 1}
	var log []routed
	if _, err := releaseHeldMessage(context.Background(), store, row, fakeEnqueuer(1, &log)); err != nil {
		t.Fatalf("release: %v", err)
	}
	if len(store.asked) != 1 {
		t.Fatalf("the release reads the subjects once, got %+v", store.asked)
	}
	got := store.asked[0]
	if !got.MaxBatchID.Valid || got.MaxBatchID.Int64 != 14 {
		t.Errorf("the read stops at the drained dispatch's last batch 14, got %+v", got.MaxBatchID)
	}
	if got.BatchID != 9 {
		t.Errorf("the lower bound stays the root's own batch (#1816), got %d", got.BatchID)
	}
}

func TestADispatchThatOpenedNoBatchBoundsNothing(t *testing.T) {
	// The reaper writes 'dead' and inserts no batch (#1391), so a hot dispatch whose every job
	// died drains with no batch of its own. Bounding at the root's own fold there would drop
	// what a later dispatch really opened, and announce an entry as nothing (ADR-1870 §3).
	// The query writes zero for that case, and for ADR-1806 §6's two degradations.
	row := heldRow(t, 33, 7)
	row.CensusUpperBatch = 0
	store := &fakeReleaseStore{claimedRows: 1}
	var log []routed
	if _, err := releaseHeldMessage(context.Background(), store, row, fakeEnqueuer(1, &log)); err != nil {
		t.Fatalf("release: %v", err)
	}
	got := store.asked[0]
	if got.MaxBatchID.Valid {
		t.Errorf("no batch of the drained dispatch is no upper bound, got %+v", got.MaxBatchID)
	}
	if got.BatchID != 7 {
		t.Errorf("the lower bound stays the root's own batch (#1816), got %d", got.BatchID)
	}
}

func TestAFailedReadReleasesNothing(t *testing.T) {
	store := &fakeReleaseStore{openedErr: errors.New("boom"), claimedRows: 1}
	var log []routed
	if _, err := releaseHeldMessage(context.Background(), store, heldRow(t, 3, 1), fakeEnqueuer(1, &log)); err == nil {
		t.Fatal("a failed read must fail the row, so its transaction rolls back")
	}
	if len(store.released) != 0 {
		t.Errorf("a census that could not be read releases nothing, got %+v", store.released)
	}
}
