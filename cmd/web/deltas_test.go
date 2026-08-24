package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// The Exposure stat band's vs-last-batch deltas (#443, P0.2): a Service that was
// firewalled at the previous batch (internet not-reached) and is exposed now
// (internet reached) moves the exposed tile +1 and the firewalled tile -1 — a real
// projection over the reachability legs as they stood then vs now, not a fabricated
// number.
func TestExposureCountDeltasAcrossBatches(t *testing.T) {
	f := newFakeStore()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	t0 := base
	t1 := base.Add(1 * time.Hour)

	const svc = "198.51.100.10:443/tcp"
	// Previous batch (t0): internal reached, internet not-reached — firewalled.
	f.addClassReachability(t, svc, "internal", t0, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", t0, `{"outcome":"not-reached"}`)
	// Latest batch (t1): internet flips to reached — now exposed (internal span holds).
	f.addClassReachability(t, svc, "internet", t1, `{"outcome":"reached"}`)

	s := newServer(f, testKey, "", func() time.Time { return t1.Add(time.Minute) })
	ctx := context.Background()

	prevAt, ok, err := s.previousBatchInstant(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a previous batch instant with two distinct batches")
	}
	if !prevAt.Equal(t0) {
		t.Fatalf("previous batch instant = %v, want %v", prevAt, t0)
	}

	exposed, firewalled, notReached, dok := s.exposureCountDeltas(ctx, prevAt)
	if !dok {
		t.Fatal("exposureCountDeltas not ok")
	}
	if exposed.Current != 1 || exposed.Previous != 0 || exposed.Change() != 1 {
		t.Errorf("exposed delta = %+v (change %d), want current 1 / previous 0 / +1", exposed, exposed.Change())
	}
	if firewalled.Current != 0 || firewalled.Previous != 1 || firewalled.Change() != -1 {
		t.Errorf("firewalled delta = %+v (change %d), want current 0 / previous 1 / -1", firewalled, firewalled.Change())
	}
	if notReached.Current != 0 || notReached.Previous != 0 {
		t.Errorf("notReached delta = %+v, want 0/0", notReached)
	}
}

// The signal deltas read the persisted first-seen (#442) to count how many of the
// currently-firing pairs were already firing at the previous batch — the honest
// net-new-since-last-batch the stored history supports. A pair first seen before the
// boundary was open then; one first seen at/after it is new.
func TestSignalDeltasNetNewSinceLastBatch(t *testing.T) {
	f := newFakeStore()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	boundary := base.Add(1 * time.Hour)
	after := base.Add(2 * time.Hour)

	// certificate-expired is critical; certificate-expiring is medium (endpoint.go).
	f.signalInstances = []db.SignalInstance{
		{ID: 1000, SignalName: "certificate-expired", SubjectKey: "a@198.51.100.1:443/tcp", FirstSeen: pgtype.Timestamptz{Time: base, Valid: true}},
		{ID: 1001, SignalName: "certificate-expired", SubjectKey: "b@198.51.100.2:443/tcp", FirstSeen: pgtype.Timestamptz{Time: after, Valid: true}},
		{ID: 1002, SignalName: "certificate-expiring", SubjectKey: "c@198.51.100.3:443/tcp", FirstSeen: pgtype.Timestamptz{Time: base, Valid: true}},
	}
	fired := []firedSignal{
		{Rule: "certificate-expired", Subject: "a@198.51.100.1:443/tcp"},
		{Rule: "certificate-expired", Subject: "b@198.51.100.2:443/tcp"},
		{Rule: "certificate-expiring", Subject: "c@198.51.100.3:443/tcp"},
	}

	s := newServer(f, testKey, "", func() time.Time { return after.Add(time.Minute) })
	open, critical, err := s.signalDeltas(context.Background(), fired, boundary)
	if err != nil {
		t.Fatal(err)
	}
	// Open: 3 firing now, 2 already firing before the boundary (a, c).
	if open.Current != 3 || open.Previous != 2 || open.Change() != 1 {
		t.Errorf("open signals delta = %+v (change %d), want 3 / 2 / +1", open, open.Change())
	}
	// Critical: a and b are critical now (2); only a was firing before the boundary (1).
	if critical.Current != 2 || critical.Previous != 1 || critical.Change() != 1 {
		t.Errorf("critical delta = %+v (change %d), want 2 / 1 / +1", critical, critical.Change())
	}
}

// With only one batch there is no previous batch to compare against, so every delta
// is withheld (Known=false) rather than compared against nothing.
func TestDeltasWithheldWithoutPreviousBatch(t *testing.T) {
	f := newFakeStore()
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", now, `{"outcome":"reached"}`)

	s := newServer(f, testKey, "", func() time.Time { return now.Add(time.Minute) })
	if _, ok, err := s.previousBatchInstant(context.Background()); err != nil || ok {
		t.Fatalf("previousBatchInstant ok=%v err=%v, want ok=false with a single batch instant", ok, err)
	}
	if d := s.dashboardDeltas(context.Background(), nil); d.Known {
		t.Errorf("dashboardDeltas Known = true, want false without a previous batch")
	}
}

// countCertsExpiring counts only certificate spans whose parsed not_after falls in
// the (ref, ref+30d] window — self-contained so it dedupes trivially with #445's
// current-state count. The shipped certificate value carries no not_after, so those
// spans (and non-certificate spans) never count.
func TestCountCertsExpiringWindow(t *testing.T) {
	ref := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	certSpan := func(subject string, notAfter string) drift.Span {
		val := `{"outcome":"presented"}`
		if notAfter != "" {
			val = fmt.Sprintf(`{"outcome":"presented","not_after":%q}`, notAfter)
		}
		return drift.Span{
			Key:   drift.TimelineKey{SubjectKind: "endpoint", SubjectKey: subject, Facet: connectoutcome.FacetCertificate},
			Value: val,
		}
	}
	// within 30d counts; beyond 30d, already expired, no not_after (the shipped
	// shape), and a non-certificate span all do not.
	spans := []drift.Span{
		certSpan("in@svc", ref.Add(10*24*time.Hour).Format(time.RFC3339)),
		certSpan("far@svc", ref.Add(40*24*time.Hour).Format(time.RFC3339)),
		certSpan("gone@svc", ref.Add(-24*time.Hour).Format(time.RFC3339)),
		certSpan("noafter@svc", ""),
		{Key: drift.TimelineKey{SubjectKind: "name", SubjectKey: "n", Facet: "resolution"}, Value: `{"outcome":"Resolved"}`},
	}
	if got := countCertsExpiring(spans, ref); got != 1 {
		t.Errorf("countCertsExpiring = %d, want 1 (only the cert expiring within 30d)", got)
	}
}
