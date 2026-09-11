package queue

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

const settleBatch = int64(70)

type fakeSettleStore struct {
	batches    []int64
	moves      []db.ListRePointMovesForBatchesRow
	citers     []db.ListResolutionCitersForAddressesAtRow
	roots      []db.ListNameRootsOpenedInBatchRow
	opened     []db.ListSubjectsOpenedSinceBatchRow
	seeds      []db.ListSeedsRow
	exclusions []db.ListExclusionsRow

	claimed    []db.ClaimSettleableRePointBatchesParams
	askedAt    []pgtype.Timestamptz
	askedMoves [][]int64
	inserted   []db.InsertMessageParams
	claimErr   error
}

func (f *fakeSettleStore) ClaimSettleableRePointBatches(_ context.Context, arg db.ClaimSettleableRePointBatchesParams) ([]int64, error) {
	f.claimed = append(f.claimed, arg)
	return f.batches, f.claimErr
}

func (f *fakeSettleStore) ListRePointMovesForBatches(_ context.Context, batchIDs []int64) ([]db.ListRePointMovesForBatchesRow, error) {
	f.askedMoves = append(f.askedMoves, batchIDs)
	return f.moves, nil
}

func (f *fakeSettleStore) ListResolutionCitersForAddressesAt(_ context.Context, arg db.ListResolutionCitersForAddressesAtParams) ([]db.ListResolutionCitersForAddressesAtRow, error) {
	f.askedAt = append(f.askedAt, arg.At)
	return f.citers, nil
}

func (f *fakeSettleStore) ListNameRootsOpenedInBatch(_ context.Context, _ int64) ([]db.ListNameRootsOpenedInBatchRow, error) {
	return f.roots, nil
}

func (f *fakeSettleStore) ListSubjectsOpenedSinceBatch(_ context.Context, _ int64) ([]db.ListSubjectsOpenedSinceBatchRow, error) {
	return f.opened, nil
}

func (f *fakeSettleStore) ListSeeds(_ context.Context) ([]db.ListSeedsRow, error) {
	return f.seeds, nil
}

func (f *fakeSettleStore) ListExclusions(_ context.Context) ([]db.ListExclusionsRow, error) {
	return f.exclusions, nil
}

func (f *fakeSettleStore) InsertMessage(_ context.Context, arg db.InsertMessageParams) (db.Message, error) {
	f.inserted = append(f.inserted, arg)
	return db.Message{ID: int64(len(f.inserted))}, nil
}

func moveRow(name string, prev, next []byte) db.ListRePointMovesForBatchesRow {
	return db.ListRePointMovesForBatchesRow{
		OpenedBatchID: pgtype.Int8{Int64: settleBatch, Valid: true},
		SubjectKey:    name,
		OpenedAt:      tstz(produceT0),
		Value:         next,
		Previous:      prev,
	}
}

func citerAt(addr, name string) db.ListResolutionCitersForAddressesAtRow {
	return db.ListResolutionCitersForAddressesAtRow{Addr: addr, SubjectKey: name}
}

func beneath(name, addr, port string) []db.ListSubjectsOpenedSinceBatchRow {
	svc := addr + ":" + port + "/tcp"
	return []db.ListSubjectsOpenedSinceBatchRow{
		subjectRow(subjectKindService, svc),
		subjectRow(subjectKindEndpoint, name+"@"+svc),
	}
}

func settleFrom(t *testing.T, store *fakeSettleStore) ([]*message.Message, []routed) {
	t.Helper()
	var log []routed
	n, err := settleRePoints(context.Background(), store, produceT0, false, fakeEnqueuer(1, &log))
	if err != nil {
		t.Fatalf("settle: %v", err)
	}
	msgs := make([]*message.Message, 0, len(store.inserted))
	for _, p := range store.inserted {
		census, err := message.ParseCensus(p.Census)
		if err != nil {
			t.Fatalf("parse census: %v", err)
		}
		c := census
		msgs = append(msgs, &message.Message{
			Cause: message.Cause(p.Cause), Class: message.Class(p.Class),
			SubjectKind: p.SubjectKind, FiredAt: p.FiredAt, Instant: p.Instant.Time,
			Census: &c, Headline: p.Headline,
		})
	}
	if n != len(msgs) {
		t.Errorf("settle reported %d message(s) and wrote %d", n, len(msgs))
	}
	return msgs, log
}

func knownAddressStore(moves ...db.ListRePointMovesForBatchesRow) *fakeSettleStore {
	return &fakeSettleStore{
		batches: []int64{settleBatch},
		moves:   moves,
		// A third Name already cites the address, so it entered the estate long ago.
		citers: []db.ListResolutionCitersForAddressesAtRow{citerAt(rpNew, "cdn.example.com")},
	}
}

func TestTheResidueFiresOnWhatTheHotTierOpenedBeneathTheNewAddress(t *testing.T) {
	// The Endpoint opens in a later hot fold, so the move's own fold counted none (#1774).
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	store.opened = beneath(rpName, rpNew, "443")

	msgs, log := settleFrom(t, store)
	if len(msgs) != 1 {
		t.Fatalf("a non-empty residue is one ADR-0026 §2 message, got %+v", msgs)
	}
	m := msgs[0]
	if m.SubjectKind != subjectKindName || m.FiredAt != rpName {
		t.Errorf("the move's Name is what fired, got %+v", m)
	}
	if m.Cause != message.CauseDrift || m.Class != message.ClassDrift {
		t.Errorf("a re-point is drift, got %+v", m)
	}
	if censusKinds(m)["endpoint"] != 1 || m.Census.Len() != 1 {
		t.Errorf("the residue is exactly the Endpoint beneath the newly cited address, got %+v", m.Census)
	}
	if !m.Instant.Equal(produceT0) {
		t.Errorf("the message carries the move's own instant, got %s", m.Instant)
	}
	if !strings.Contains(m.Headline, "re-pointed within the estate") {
		t.Errorf("headline = %q", m.Headline)
	}
	if len(log) != 1 || log[0].class != message.ClassDrift {
		t.Errorf("a fired message is routed once, got %+v", log)
	}
}

func TestTheResidueTakesNoHeldRow(t *testing.T) {
	// Membership owes a held row because its root determination is fold-local. A move's
	// predicate reads two durable spans, so it owes none (ADR-1806 §2).
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	store.opened = beneath(rpName, rpNew, "443")

	settleFrom(t, store)
	p := store.inserted[0]
	if p.CensusPendingAfterBatch.Valid || len(p.CensusBasis) != 0 {
		t.Errorf("the re-point path writes no held row, got %+v", p)
	}
}

func TestAnEmptyResidueFiresNothing(t *testing.T) {
	// A move that opened nothing beneath its new address says nothing to the operator.
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))

	msgs, log := settleFrom(t, store)
	if len(msgs) != 0 || len(log) != 0 {
		t.Fatalf("an empty residue is not a firing (ADR-0026 §2), got %+v", msgs)
	}
}

func TestASettledFoldIsReadNoSecondTime(t *testing.T) {
	// The guarded UPDATE is the claim, so the pass that arrives second takes no batch and the
	// minute poll cannot re-announce a move it already fired (ADR-1806 §8).
	store := &fakeSettleStore{opened: beneath(rpName, rpNew, "443")}

	msgs, log := settleFrom(t, store)
	if len(msgs) != 0 || len(log) != 0 {
		t.Fatalf("a pass that claimed no fold writes nothing, got %+v", msgs)
	}
	if len(store.askedMoves) != 0 {
		t.Errorf("a pass that claimed no fold reads no move, got %v", store.askedMoves)
	}
}

func TestTheClaimCarriesTheSameDegradationTheReleaseReads(t *testing.T) {
	// One tier, one bound: a reaper the operator disabled leaves the drain test unable to
	// conclude on both paths at once (ADR-1806 §6).
	store := &fakeSettleStore{}
	var log []routed
	if _, err := settleRePoints(context.Background(), store, produceT0, true, fakeEnqueuer(1, &log)); err != nil {
		t.Fatalf("settle: %v", err)
	}
	if len(store.claimed) != 1 || !store.claimed[0].ReaperDisabled {
		t.Errorf("the caller passes the reaper's state through, got %+v", store.claimed)
	}
	if !store.claimed[0].SettledAt.Time.Equal(produceT0) {
		t.Errorf("the settled instant is the caller's clock, got %+v", store.claimed[0].SettledAt)
	}
}

func TestAnAddressNewToTheEstateLeavesTheResidueEmpty(t *testing.T) {
	// The Address root covers the whole residue, so no second message fires (ADR-0026 §2).
	store := &fakeSettleStore{
		batches: []int64{settleBatch},
		moves:   []db.ListRePointMovesForBatchesRow{moveRow(rpName, resolved(rpOld), resolved(rpNew))},
		opened:  beneath(rpName, rpNew, "443"),
	}
	msgs, _ := settleFrom(t, store)
	if len(msgs) != 0 {
		t.Fatalf("the Address root and the residue partition one ground, got %+v", msgs)
	}
}

func TestTheResidueDropsWhatAMembershipRootOfTheSameFoldCovers(t *testing.T) {
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	store.opened = beneath(rpOther, rpNew, "443")
	store.roots = []db.ListNameRootsOpenedInBatchRow{{SubjectKey: rpOther, Value: resolved(rpNew)}}

	msgs, _ := settleFrom(t, store)
	if len(msgs) != 0 {
		t.Fatalf("a membership message of the same fold already covers it (ADR-0026 §2), got %+v", msgs)
	}
}

func TestTheResidueIsOnlyBeneathTheNewlyCitedAddresses(t *testing.T) {
	// The move keeps rpOld and adds rpNew, which another Name already cites.
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpOld, rpNew)))
	store.opened = append(beneath(rpName, rpOld, "8443"), beneath(rpName, rpNew, "443")...)

	msgs, _ := settleFrom(t, store)
	if len(msgs) != 1 || msgs[0].Census.Len() != 1 {
		t.Fatalf("a new port on an address the Name already cited is not the move's consequence; got %+v", msgs)
	}
	if e := msgs[0].Census.Entries[0]; !strings.Contains(e.Key, rpNew) {
		t.Errorf("the residue is the Endpoint beneath the newly cited address, got %+v", e)
	}
}

func TestASeedCoveredAddressIsNoRootSoTheEndpointIsResidue(t *testing.T) {
	store := &fakeSettleStore{
		batches: []int64{settleBatch},
		moves:   []db.ListRePointMovesForBatchesRow{moveRow(rpName, resolved(rpOld), resolved(rpNew))},
		opened:  beneath(rpName, rpNew, "443"),
		seeds:   []db.ListSeedsRow{addressSeed("203.0.113.0/24")},
	}
	msgs, _ := settleFrom(t, store)
	if len(msgs) != 1 || censusKinds(msgs[0])["endpoint"] != 1 {
		t.Fatalf("a Seed-covered address never appears (ADR-0047), so the Endpoint is residue; got %+v", msgs)
	}
}

func TestEachMoveOfAFoldIsItsOwnCause(t *testing.T) {
	store := knownAddressStore(
		moveRow(rpName, resolved(rpOld), resolved(rpNew)),
		moveRow(rpOther, resolved(rpOld), resolved(rpNew)),
	)
	store.opened = beneath("", rpNew, "443")

	msgs, log := settleFrom(t, store)
	if len(msgs) != 2 {
		t.Fatalf("each Name's move is its own cause, so each fires (ADR-0026 §2); got %+v", msgs)
	}
	for _, m := range msgs {
		if censusKinds(m)["endpoint"] != 1 {
			t.Errorf("both Names gained the dispatcher's ground, so both residues name it, got %+v", m.Census)
		}
	}
	if len(log) != 2 {
		t.Errorf("each message is routed once, got %+v", log)
	}
}

func TestAFoldThatGainedNoAddressReadsNoEstate(t *testing.T) {
	// Two Names swapped their addresses, so the fold as a whole cited nothing new. The residue
	// is still each move's own, and no address is a root candidate (#1730).
	store := &fakeSettleStore{
		batches: []int64{settleBatch},
		moves: []db.ListRePointMovesForBatchesRow{
			moveRow(rpName, resolved(rpOld), resolved(rpNew)),
			moveRow(rpOther, resolved(rpNew), resolved(rpOld)),
		},
		opened: beneath(rpName, rpNew, "443"),
	}
	msgs, _ := settleFrom(t, store)
	if len(store.askedAt) != 0 {
		t.Errorf("no candidate address means no estate read, got %v", store.askedAt)
	}
	if len(msgs) != 1 || msgs[0].FiredAt != rpName {
		t.Fatalf("an address a sibling just dropped was in the estate, so the move is residue; got %+v", msgs)
	}
}

func TestTheCiterReadIsTakenAtTheMovesOwnInstant(t *testing.T) {
	// Live resolution would fold a later move's citers into this one's freshness test (§4).
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	settleFrom(t, store)
	if len(store.askedAt) != 1 || !store.askedAt[0].Time.Equal(produceT0) {
		t.Errorf("the read is bounded at the fold's own instant, got %+v", store.askedAt)
	}
}

func TestAFailedClaimSettlesNothing(t *testing.T) {
	store := &fakeSettleStore{claimErr: errors.New("boom")}
	var log []routed
	if _, err := settleRePoints(context.Background(), store, produceT0, false, fakeEnqueuer(1, &log)); err == nil {
		t.Fatal("a failed claim must fail the pass, so its transaction rolls back")
	}
	if len(store.inserted) != 0 {
		t.Errorf("a pass that claimed nothing writes nothing, got %+v", store.inserted)
	}
}
