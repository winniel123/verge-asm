package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

	s := &server{deltasStore: f, vantageClassStore: f, now: func() time.Time { return t1.Add(time.Minute) }}
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

	s := &server{deltasStore: f, now: func() time.Time { return after.Add(time.Minute) }}
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

	s := &server{deltasStore: f, vantageClassStore: f, now: func() time.Time { return now.Add(time.Minute) }}
	if _, ok, err := s.previousBatchInstant(context.Background()); err != nil || ok {
		t.Fatalf("previousBatchInstant ok=%v err=%v, want ok=false with a single batch instant", ok, err)
	}
	if d := s.dashboardDeltas(context.Background(), nil); d.Known {
		t.Errorf("dashboardDeltas Known = true, want false without a previous batch")
	}
}

func TestCountCertsExpiringWindow(t *testing.T) {
	// The window is a third of each certificate's own validity,
	// a half below ten days (ADR-0004 #67).
	ref := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	certSpan := func(subject string, notBefore, notAfter time.Time) drift.Span {
		val := `{"outcome":"presented"}`
		switch {
		case !notAfter.IsZero() && !notBefore.IsZero():
			val = fmt.Sprintf(`{"outcome":"presented","not_before":%q,"not_after":%q}`, notBefore.Format(time.RFC3339), notAfter.Format(time.RFC3339))
		case !notAfter.IsZero():
			val = fmt.Sprintf(`{"outcome":"presented","not_after":%q}`, notAfter.Format(time.RFC3339))
		}
		return drift.Span{
			Key:   drift.TimelineKey{SubjectKind: "endpoint", SubjectKey: subject, Facet: connectoutcome.FacetCertificate},
			Value: val,
		}
	}
	spans := []drift.Span{
		certSpan("in@svc", ref.Add(-80*day), ref.Add(10*day)),
		certSpan("long-in@svc", ref.Add(-298*day), ref.Add(100*day)),
		certSpan("far@svc", ref.Add(-50*day), ref.Add(40*day)),
		certSpan("six-day-fresh@svc", ref.Add(-1*day), ref.Add(5*day)),
		certSpan("six-day-in@svc", ref.Add(-4*day), ref.Add(2*day)),
		certSpan("gone@svc", ref.Add(-90*day), ref.Add(-1*day)),
		certSpan("nobefore@svc", time.Time{}, ref.Add(10*day)),
		certSpan("noafter@svc", time.Time{}, time.Time{}),
		{Key: drift.TimelineKey{SubjectKind: "name", SubjectKey: "n", Facet: "resolution"}, Value: `{"outcome":"Resolved"}`},
	}
	if got := countCertsExpiring(spans, ref); got != 3 {
		t.Errorf("countCertsExpiring = %d, want 3 (in, long-in, six-day-in)", got)
	}
}

func (f *fakeStore) PreviousBatchTime(_ context.Context) (pgtype.Timestamptz, error) {
	inst := map[int64]time.Time{}
	for _, o := range f.observations {
		if t := o.ObservedAt.Time; t.After(inst[o.BatchID]) {
			inst[o.BatchID] = t
		}
	}
	var latest, prev time.Time
	for _, t := range inst {
		switch {
		case t.After(latest):
			prev = latest
			latest = t
		case t.Before(latest) && t.After(prev):
			prev = t
		}
	}
	if prev.IsZero() {
		return pgtype.Timestamptz{}, nil
	}
	return pgtype.Timestamptz{Time: prev, Valid: true}, nil
}

func (f *fakeStore) ListSpansOpenSince(_ context.Context, since pgtype.Timestamptz) ([]db.ListSpansOpenSinceRow, error) {
	type tlkey struct{ kind, key, facet, discriminator, source string }
	order := []tlkey{}
	byKey := map[tlkey][]drift.Reading{}
	for _, o := range f.observations {
		k := tlkey{o.SubjectKind, o.SubjectKey, o.Facet, o.Discriminator, o.Source}
		if _, seen := byKey[k]; !seen {
			order = append(order, k)
		}
		gap := o.Facet == "resolution" && fakeResolutionOutcome(o.Value) == "Gap"
		if o.Facet == "reachability" {
			gap = reachOutcomeIsGap(o.Value)
		}
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

	rows := []db.ListSpansOpenSinceRow{}
	var id int64
	for _, k := range order {
		derivation, _ := json.Marshal(fakeFacetVector(k.facet))
		key := drift.TimelineKey{
			SubjectKind: k.kind, SubjectKey: k.key,
			Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
		}
		for _, s := range drift.Fold(key, byKey[k]) {
			if !s.ClosedAt.IsZero() && !s.ClosedAt.After(since.Time) {
				continue
			}
			id++
			row := db.ListSpansOpenSinceRow{
				ID: id, SubjectKind: k.kind, SubjectKey: k.key,
				Facet: k.facet, Discriminator: k.discriminator, Source: k.source,
				Value: []byte(s.Value), IsGap: s.IsGap, Derivation: derivation,
				OpenedAt: pgtype.Timestamptz{Time: s.OpenedAt, Valid: true},
			}
			if !s.ClosedAt.IsZero() {
				row.ClosedAt = pgtype.Timestamptz{Time: s.ClosedAt, Valid: true}
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (f *fakeStore) ListServiceReachabilitySpansByClassAt(_ context.Context, at pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error) {
	known := map[int64]bool{}
	for _, v := range f.vantages {
		known[v.ID] = true
	}
	type tlkey struct {
		svc     string
		vantage int64
	}
	byKey := map[tlkey][]drift.Reading{}
	for _, o := range f.observations {
		if o.SubjectKind != "service" || o.Facet != "reachability" || !o.VantageID.Valid {
			continue
		}
		if !known[o.VantageID.Int64] {
			continue
		}
		k := tlkey{o.SubjectKey, o.VantageID.Int64}
		byKey[k] = append(byKey[k], drift.Reading{
			Value: string(o.Value), IsGap: reachOutcomeIsGap(o.Value),
			Vector: fakeFacetVector("reachability"), ObservedAt: o.ObservedAt.Time,
		})
	}

	rows := []db.ListServiceReachabilitySpansByClassAtRow{}
	for k, readings := range byKey {
		key := drift.TimelineKey{SubjectKind: "service", SubjectKey: k.svc, Facet: "reachability"}
		var chosen *drift.Span
		for _, s := range drift.Fold(key, readings) {
			if s.OpenedAt.After(at.Time) {
				continue
			}
			if !s.ClosedAt.IsZero() && !s.ClosedAt.After(at.Time) {
				continue
			}
			if chosen == nil || s.OpenedAt.After(chosen.OpenedAt) {
				sp := s
				chosen = &sp
			}
		}
		if chosen == nil {
			continue
		}
		v := f.vantageByID(k.vantage)
		rows = append(rows, db.ListServiceReachabilitySpansByClassAtRow{
			SubjectKey: k.svc, VantageID: pgtype.Int8{Int64: k.vantage, Valid: true},
			Value: []byte(chosen.Value), IsGap: chosen.IsGap,
			OpenedAt: pgtype.Timestamptz{Time: chosen.OpenedAt, Valid: true}, ID: k.vantage,
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
