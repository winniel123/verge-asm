package main

import (
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/exposure"
)

func exposureFoldFixture(t *testing.T, f *fakeStore, at time.Time) {
	t.Helper()
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", at, `{"outcome":"reached"}`)
	// Edge-only: a DMZ Service reached from the internet and not reached inside.
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internal", at, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internet", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.12:5432/tcp", "internal", at, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.12:5432/tcp", "internet", at, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, "198.51.100.13:8080/tcp", "internal", at, `{"outcome":"not-reached"}`)
	f.addClassReachability(t, "198.51.100.13:8080/tcp", "internet", at, `{"outcome":"not-reached"}`)
	// One-legged: no internet leg was ever configured for this Service.
	f.addClassReachability(t, "198.51.100.14:80/tcp", "internal", at, `{"outcome":"reached"}`)
}

func foldExposureForTest(t *testing.T, f *fakeStore) ([]exposureRow, exposure.Census) {
	t.Helper()
	s := &server{exposureStore: f, vantageClassStore: f}
	covered, err := s.addressScopeCovered(t.Context())
	if err != nil {
		t.Fatalf("addressScopeCovered: %v", err)
	}
	rows, census, err := s.foldExposureUnder(t.Context(), covered)
	if err != nil {
		t.Fatalf("foldExposureUnder: %v", err)
	}
	return rows, census
}

// Every row lands in a cell, or the band's figures are a claim about an unstated denominator
// (#1966).

func TestExposureBandAccountsForEveryRow(t *testing.T) {
	f := newFakeStore()
	exposureFoldFixture(t, f, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))

	rows, census := foldExposureForTest(t, f)
	want := exposure.Census{Exposed: 1, EdgeOnly: 1, Firewalled: 1, Unreachable: 1, OneLegged: 1}
	if census != want {
		t.Errorf("census = %+v, want %+v", census, want)
	}
	if census.Total() != len(rows) {
		t.Errorf("the band counted %d of %d rows; an uncounted row renders a chip under zeroes",
			census.Total(), len(rows))
	}
}

// A leg's date is read from the open spans of that leg's own Vantage class that carry the leg's
// value, so the two legs of one row may carry two dates (#2034). Which span of the class dates
// the leg is #2059's rule, and legFromClassGroup holds it; this fixture seeds one span per class
// and separates the classes alone.

func TestExposureLegsCarryTheirOwnDate(t *testing.T) {
	f := newFakeStore()
	early := time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)
	late := time.Date(2026, 8, 3, 9, 30, 0, 0, time.UTC)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internal", early, `{"outcome":"reached"}`)
	f.addClassReachability(t, "198.51.100.10:443/tcp", "internet", late, `{"outcome":"reached"}`)
	// A leg that was never configured holds no value for a date to belong to.
	f.addClassReachability(t, "198.51.100.11:22/tcp", "internal", early, `{"outcome":"reached"}`)

	rows, _ := foldExposureForTest(t, f)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if got, want := rows[0].Internal.Date, "2026-08-01"; got != want {
		t.Errorf("internal leg date = %q, want %q", got, want)
	}
	if got, want := rows[0].Internet.Date, "2026-08-03"; got != want {
		t.Errorf("internet leg date = %q, want %q; a class-blind date sits beside the wrong leg", got, want)
	}
	if got := rows[1].Internet.Date; got != "" {
		t.Errorf("a never-configured leg carried the date %q", got)
	}
}
