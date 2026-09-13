package main

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
)

func (f *fakeStore) addVantagePresenting(name, dialled string) int64 {
	v := db.Vantage{
		ID: f.vantageNextID, Name: name, Class: "unverified",
		DialledAddr: pgtype.Text{String: dialled, Valid: true},
	}
	f.vantages = append(f.vantages, v)
	f.vantageNextID++
	return v.ID
}

func (f *fakeStore) addReachabilityAtVantage(t *testing.T, serviceKey string, vantageID int64, at time.Time, value string) {
	t.Helper()
	b := f.freshBatch("hot", "connect-outcome")
	f.observations = append(f.observations, db.Observation{
		ID: f.obsNextID, BatchID: b, Facet: "reachability", SubjectKind: "service",
		SubjectKey: serviceKey, VantageID: pgtype.Int8{Int64: vantageID, Valid: true},
		Source: "prober", Value: []byte(value),
		ObservedAt: pgtype.Timestamptz{Time: at, Valid: true},
	})
	f.obsNextID++
}

func (f *fakeStore) declareAddressScope(t *testing.T, cidr string) {
	t.Helper()
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		t.Fatalf("parse %q: %v", cidr, err)
	}
	f.seeds = append(f.seeds, db.Seed{ID: f.seedNextID, Kind: "address", AddressCidr: &p})
	f.seedNextID++
}

func TestAScopeEditMovesBothExposureDeltaLegsTogether(t *testing.T) {
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	t0, t1 := base, base.Add(time.Hour)
	const svc = "203.0.113.10:443/tcp"

	f := newFakeStore()
	outside := f.addVantagePresenting("outside", "198.51.100.200")
	edited := f.addVantagePresenting("edited", "192.0.2.7")
	for _, at := range []time.Time{t0, t1} {
		// The same outcome at both instants, so Current equals Previous throughout (ADR-1895 §4).
		f.addReachabilityAtVantage(t, svc, outside, at, `{"outcome":"not-reached"}`)
		f.addReachabilityAtVantage(t, svc, edited, at, `{"outcome":"reached"}`)
	}

	s := &server{deltasStore: f, vantageClassStore: f}
	ctx := context.Background()

	exposed, firewalled, notReached, ok := s.exposureCountDeltas(ctx, t0)
	if !ok {
		t.Fatal("exposureCountDeltas before the scope edit: not ok")
	}
	wantBefore := []struct {
		name string
		got  drift.Delta
		want drift.Delta
	}{
		{"exposed", exposed, drift.Delta{}},
		{"firewalled", firewalled, drift.Delta{}},
		// Both presented addresses are uncovered, so one class holds both legs and no
		// Exposure value is readable (ADR-0017).
		{"not-reached", notReached, drift.Delta{Current: 1, Previous: 1}},
	}
	for _, c := range wantBefore {
		if c.got != c.want {
			t.Errorf("before the scope edit: %s = %+v, want %+v", c.name, c.got, c.want)
		}
	}

	f.declareAddressScope(t, "192.0.2.0/24")

	exposed, firewalled, notReached, ok = s.exposureCountDeltas(ctx, t0)
	if !ok {
		t.Fatal("exposureCountDeltas after the scope edit: not ok")
	}
	wantAfter := []struct {
		name string
		got  drift.Delta
		want drift.Delta
	}{
		{"exposed", exposed, drift.Delta{}},
		{"firewalled", firewalled, drift.Delta{Current: 1, Previous: 1}},
		{"not-reached", notReached, drift.Delta{}},
	}
	// Every want above holds Current == Previous, so the equality check is the invariant too.
	for _, c := range wantAfter {
		if c.got != c.want {
			t.Errorf("after the scope edit: %s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
}
