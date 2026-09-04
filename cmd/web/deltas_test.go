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

func TestExposureCountDeltasAcrossBatches(t *testing.T) {
	f := newFakeStore()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	t0 := base
	t1 := base.Add(1 * time.Hour)

	const svc = "198.51.100.10:443/tcp"
	f.addClassReachability(t, svc, "internal", t0, `{"outcome":"reached"}`)
	f.addClassReachability(t, svc, "internet", t0, `{"outcome":"not-reached"}`)
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

func TestSignalDeltasNetNewSinceLastBatch(t *testing.T) {
	f := newFakeStore()
	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	boundary := base.Add(1 * time.Hour)
	after := base.Add(2 * time.Hour)

	// certificate-expired is critical and certificate-expiring medium (internal/signal).
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
	if open.Current != 3 || open.Previous != 2 || open.Change() != 1 {
		t.Errorf("open signals delta = %+v (change %d), want 3 / 2 / +1", open, open.Change())
	}
	if critical.Current != 2 || critical.Previous != 1 || critical.Change() != 1 {
		t.Errorf("critical delta = %+v (change %d), want 2 / 1 / +1", critical, critical.Change())
	}
}

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
