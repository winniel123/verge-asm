package queue

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/scan"
)

func reachOpening(svc string) spanChange {
	return spanChange{
		SubjectKind: "service", SubjectKey: svc, Facet: "reachability",
		Opened: true, Value: reachValue("reached"),
	}
}

func reachRowFor(l classLeg) db.ListServiceReachabilitySpansByClassForServicesRow {
	dialled, vantage := dialledInternal, int64(2)
	if l.class == "internet" {
		dialled, vantage = dialledInternet, 1
	}
	return db.ListServiceReachabilitySpansByClassForServicesRow{
		SubjectKey: l.subject, VantageID: pgtype.Int8{Int64: vantage, Valid: true},
		Value: reachValue(l.outcome), DialledAddr: pgtype.Text{String: dialled, Valid: true},
	}
}

func widenStore(cur, prev []classLeg) *fakeMessageStore {
	f := &fakeMessageStore{
		prev: pgtype.Timestamptz{Time: produceT0.Add(-time.Hour), Valid: true},
		vantages: []db.ListVantagesForDispatchRow{
			{ID: 1, DialledAddr: pgtype.Text{String: dialledInternet, Valid: true}},
			{ID: 2, DialledAddr: pgtype.Text{String: dialledInternal, Valid: true}},
		},
	}
	for _, l := range cur {
		f.current = append(f.current, reachRowFor(l))
	}
	for _, l := range prev {
		f.at = append(f.at, db.ListServiceReachabilitySpansByClassAtForServicesRow(reachRowFor(l)))
	}
	return f
}

func wideningMessages(store *fakeMessageStore) []db.InsertMessageParams {
	var out []db.InsertMessageParams
	for _, m := range store.inserted {
		if m.SubjectKind == message.KindVantageClass {
			out = append(out, m)
		}
	}
	return out
}

func TestVantageClassWideningFiresOnceAtTheClassWithTheFoldsCensus(t *testing.T) {
	const svc1, svc2 = "10.1.0.1:443/tcp", "10.1.0.2:443/tcp"
	store := widenStore(
		[]classLeg{
			{subject: svc1, class: "internal", outcome: "reached"},
			{subject: svc1, class: "internet", outcome: "reached"},
			{subject: svc2, class: "internal", outcome: "reached"},
			{subject: svc2, class: "internet", outcome: "not-reached"},
		},
		[]classLeg{
			{subject: svc1, class: "internal", outcome: "reached"},
			{subject: svc2, class: "internal", outcome: "reached"},
		})
	store.classUnfolded = true
	changes := []spanChange{reachOpening(svc1), reachOpening(svc2)}
	var log []routed
	if err := produceMessages(context.Background(), store, 40, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	widen := wideningMessages(store)
	if len(widen) != 1 {
		t.Fatalf("one message per widened class, got %d of %d: %+v", len(widen), len(store.inserted), store.inserted)
	}
	m := widen[0]
	if m.Cause != string(message.CauseAperture) || m.Class != string(message.ClassCoverage) {
		t.Errorf("a widening is coverage class, got cause=%q class=%q", m.Cause, m.Class)
	}
	if m.FiredAt != "internet" {
		t.Errorf("fires at the class that arrived, got %q", m.FiredAt)
	}
	c, _ := message.ParseCensus(m.Census)
	if c.Len() != 2 || c.Entries[0].Key != svc1 || c.Entries[0].Detail != "exposed" || c.Entries[1].Detail != "firewalled" {
		t.Errorf("the census is the fold's Services with their composed value, got %+v", c.Entries)
	}
	if !strings.Contains(m.Headline, "an internet vantage is now configured") {
		t.Errorf("headline = %q", m.Headline)
	}
	if len(log) == 0 {
		t.Error("the widening routes on the coverage class")
	}
	if len(store.foldedAsked) != 1 {
		t.Fatalf("one batch-record read per candidate class, got %d", len(store.foldedAsked))
	}
	asked := store.foldedAsked[0]
	if asked.BeforeBatchID != 40 || len(asked.VantageIds) != 1 || asked.VantageIds[0] != 1 {
		t.Errorf("asks whether this class's vantages folded before this batch, got %+v", asked)
	}
	if len(asked.Kinds) != 2 || asked.Kinds[0] != scan.HotKind || asked.Kinds[1] != scan.ColdKind {
		t.Errorf("only the port tiers open a Reach leg, got kinds %v", asked.Kinds)
	}
}

func TestVantageClassWideningIsSilentWhereTheClassAlreadyFolded(t *testing.T) {
	const svc = "10.1.0.9:443/tcp"
	// A fresh Service has no prev leg, so the bounded read shortlists its class (ADR-0226).
	store := widenStore([]classLeg{{subject: svc, class: "internet", outcome: "reached"}}, nil)
	if err := produceMessages(context.Background(), store, 41, produceT0, []spanChange{reachOpening(svc)}, nil, nil, membershipInputs{}, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if len(store.foldedAsked) != 1 {
		t.Fatalf("the batch record settles a shortlisted class, got %d reads", len(store.foldedAsked))
	}
	if widen := wideningMessages(store); len(widen) != 0 {
		t.Errorf("a class that folded before this batch widens nothing, got %+v", widen)
	}
}

func TestVantageClassWideningReadsNothingWhereNoClassIsNew(t *testing.T) {
	const svc = "10.1.0.1:443/tcp"
	legs := []classLeg{{subject: svc, class: "internet", outcome: "not-reached"}}
	store := widenStore(legs, legs)
	if err := produceMessages(context.Background(), store, 40, produceT0, []spanChange{reachOpening(svc)}, nil, nil, membershipInputs{}, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if store.vantageReads != 0 || len(store.foldedAsked) != 0 {
		t.Errorf("a class the bounded legs already held costs no further read, got %d/%d", store.vantageReads, len(store.foldedAsked))
	}
}

func TestVantageClassWideningNeedsAReachabilityOpening(t *testing.T) {
	store := widenStore([]classLeg{{subject: "10.1.0.1:443/tcp", class: "internet", outcome: "reached"}}, nil)
	store.classUnfolded = true
	changes := []spanChange{{SubjectKind: "name", SubjectKey: "www.example.com", Facet: "resolution", Opened: true}}
	if err := produceMessages(context.Background(), store, 40, produceT0, changes, nil, nil, membershipInputs{}, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	if store.vantageReads != 0 {
		t.Error("a fold that opened no Reach leg asks nothing")
	}
	if widen := wideningMessages(store); len(widen) != 0 {
		t.Errorf("a fold that opened no Reach leg widened no class, got %+v", widen)
	}
}

func TestVantageClassWideningStatesAnEmptyCensus(t *testing.T) {
	const svc = "10.1.0.1:443/tcp"
	// The first class to run composes nothing, because Exposure needs both legs (ADR-0017).
	store := widenStore([]classLeg{{subject: svc, class: "internal", outcome: "reached"}}, nil)
	store.prev = pgtype.Timestamptz{}
	store.classUnfolded = true
	if err := produceMessages(context.Background(), store, 1, produceT0, []spanChange{reachOpening(svc)}, nil, nil, membershipInputs{}, nil, false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	widen := wideningMessages(store)
	if len(widen) != 1 || widen[0].FiredAt != "internal" {
		t.Fatalf("the first class fires at itself, got %+v", widen)
	}
	if !strings.HasSuffix(widen[0].Headline, "no Exposure timeline opened") {
		t.Errorf("headline = %q, want the zero stated", widen[0].Headline)
	}
}
