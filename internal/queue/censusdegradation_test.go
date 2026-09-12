package queue

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

func TestNoReaperWritesTheCensusAtTheCause(t *testing.T) {
	// The drain test reads a job set nothing reaps, so a hold would never release (ADR-1806 §6).
	changes, store := batchMovingBothSignals()
	var routedLog []routed

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &routedLog), false, false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	var membership *db.InsertMessageParams
	for i := range store.inserted {
		if store.inserted[i].SubjectKind == "name" {
			membership = &store.inserted[i]
		}
	}
	if membership == nil {
		t.Fatal("no membership message written")
	}
	if membership.CensusPendingAfterBatch.Valid {
		t.Errorf("no reaper means no hold, got a row held on batch %d", membership.CensusPendingAfterBatch.Int64)
	}
	if len(membership.CensusBasis) != 0 {
		t.Errorf("a row that is not held owes no frozen basis, got %s", membership.CensusBasis)
	}
	census, err := message.ParseCensus(membership.Census)
	if err != nil {
		t.Fatalf("parse census: %v", err)
	}
	// The fold's own openings are all this census can ever hold (ADR-1806 §6).
	if census.Len() != 2 || census.Entries[0].Key != "example.com@198.51.100.1:443/tcp" || census.Entries[1].Key != "198.51.100.1:443/tcp" {
		t.Errorf("the census counts what this fold opened beneath the root, got %+v", census.Entries)
	}
	if membership.Headline != "example.com entered the estate · 1 endpoint + 1 service · 2 timelines opened on an address it cites" {
		t.Errorf("the headline carries the cause clause and the census clause, got %q", membership.Headline)
	}
}

func TestNoReaperRoutesTheMembershipAtTheCause(t *testing.T) {
	// Only the release poll routes a held row, so an unheld row owes its own delivery (§6).
	changes, store := batchMovingBothSignals()
	var routedLog []routed

	if err := produceMessages(context.Background(), store, 7, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &routedLog), false, false); err != nil {
		t.Fatalf("produce: %v", err)
	}

	want := insertedID(store, "name")
	found := false
	for _, r := range routedLog {
		if r.messageID == want {
			found = true
		}
	}
	if !found {
		t.Fatalf("the membership message is routed at the cause, got %+v", routedLog)
	}
}

func TestTheUnarmedGateWarningNamesTheCensusLoss(t *testing.T) {
	var buf bytes.Buffer
	// The warning spoke about a doubled probe rate alone, so the census loss was silent (§6).
	if _, err := hotTickLags(context.Background(), &fakeHotLagStore{}, 7, 42, 0, log.New(&buf, "", 0)); err != nil {
		t.Fatalf("hotTickLags: %v", err)
	}
	for _, want := range []string{"census", "its own fold"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("the warning must name the census loss with %q, logged %q", want, buf.String())
		}
	}
}

func TestTheWorkerHoldsExactlyWhileTheReaperRuns(t *testing.T) {
	// The fold that writes a hold and the poll that releases it read one knob (#1114).
	w := &Worker{staleJobThreshold: DefaultStaleJobThreshold}
	if !HotLagGateArmed(w.staleJobThreshold) {
		t.Error("the default stale-job timeout runs the reaper, so the fold holds its census")
	}
	w.WithStaleJobThreshold(0)
	if HotLagGateArmed(w.staleJobThreshold) {
		t.Error("a zero stale-job timeout disables the reaper, so the fold may write no hold")
	}
	w.WithStaleJobThreshold(-time.Minute)
	if HotLagGateArmed(w.staleJobThreshold) {
		t.Error("a negative stale-job timeout disables the reaper, so the fold may write no hold")
	}
}
