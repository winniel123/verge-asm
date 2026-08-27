package main

import (
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// P0.13 (#687): the live address-scope numerator and the per-zone stale callout that
// replace cold.go's hardcoded placeholders. These exercise the pure derivations the
// handler wires; the devMode fixture path (G2) is unchanged.

func mustPrefix(t *testing.T, s string) netip.Prefix {
	t.Helper()
	p, err := netip.ParsePrefix(s)
	if err != nil {
		t.Fatalf("parse prefix %q: %v", s, err)
	}
	return p.Masked()
}

func mustAddr(t *testing.T, s string) netip.Addr {
	t.Helper()
	a, err := netip.ParseAddr(s)
	if err != nil {
		t.Fatalf("parse addr %q: %v", s, err)
	}
	return a
}

// walkedAddresses draws only the IP-hosted Service keys, dropping name-hosted ones —
// a range's numerator counts addresses actually reached, never a name.
func TestWalkedAddressesDropsNameHosts(t *testing.T) {
	rows := []db.ListCurrentServiceSubjectsRow{
		{SubjectKey: "203.0.113.44:22"},
		{SubjectKey: "203.0.113.9:443/tcp"},
		{SubjectKey: "mail.acmecorp.io:25"}, // name host — no address
		{SubjectKey: "not-an-ip"},           // unparseable — dropped
		{SubjectKey: "198.51.100.7:80"},
	}
	got := walkedAddresses(rows)
	if len(got) != 3 {
		t.Fatalf("walkedAddresses: want 3 IP hosts, got %d (%v)", len(got), got)
	}
}

// coveredInRange counts distinct in-range addresses only — two Services on one
// address credit the range once, and out-of-range addresses do not count.
func TestCoveredInRangeDistinctAndBounded(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	walked := []netip.Addr{
		mustAddr(t, "203.0.113.44"), // in
		mustAddr(t, "203.0.113.44"), // in, duplicate — counts once
		mustAddr(t, "203.0.113.9"),  // in
		mustAddr(t, "198.51.100.7"), // out of range
		mustAddr(t, "2001:db8::1"),  // other family — never contained
	}
	if got := coveredInRange(walked, p); got != 2 {
		t.Fatalf("coveredInRange: want 2 distinct in-range, got %d", got)
	}
	if got := coveredInRange(nil, p); got != 0 {
		t.Fatalf("coveredInRange(nil): want 0, got %d", got)
	}
}

// An address scope renders the #19c counted/total meter: covered subjects over the
// range's enumerable addresses, with the ruled coveragePct fill.
func TestApertureMetersAddressCountedTotal(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	walked := []netip.Addr{
		mustAddr(t, "203.0.113.10"),
		mustAddr(t, "203.0.113.20"),
		mustAddr(t, "203.0.113.30"),
	}
	m := apertureMeters(seeds, nil, walked)
	if len(m) != 1 {
		t.Fatalf("want 1 meter, got %d", len(m))
	}
	got := m[0]
	if got.Total == nil {
		t.Fatalf("address meter should carry a denominator (counted/total), got census")
	}
	if *got.Total != "256" {
		t.Fatalf("denominator = enumerable addresses of /24: want 256, got %q", *got.Total)
	}
	if got.Counted != "3" {
		t.Fatalf("numerator = covered subjects: want 3, got %q", got.Counted)
	}
	if got.Pct != coveragePct(3, 256) {
		t.Fatalf("fill must be the ruled coveragePct(3,256)=%d, got %d", coveragePct(3, 256), got.Pct)
	}
	if got.Label != "203.0.113.0/24" {
		t.Fatalf("label = the range: got %q", got.Label)
	}
}

// A zero numerator is honest (nothing walked yet), never suppressed to a census.
func TestApertureMetersAddressZeroNumerator(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	m := apertureMeters(seeds, nil, nil)
	if m[0].Total == nil || m[0].Counted != "0" || m[0].Pct != 0 {
		t.Fatalf("empty walk: want 0/256 at 0%%, got counted=%q total=%v pct=%d", m[0].Counted, m[0].Total, m[0].Pct)
	}
}

// A name scope stays a census — no denominator (ADR-0072).
func TestApertureMetersNameCensus(t *testing.T) {
	seeds := []db.ListSeedsRow{{Kind: "name", NameDomain: pgtype.Text{String: "acmecorp.io", Valid: true}}}
	m := apertureMeters(seeds, nil, nil)
	if m[0].Total != nil {
		t.Fatalf("name scope must be a census (Total nil), got %v", m[0].Total)
	}
}

func staleRow(seedID int64, domain string, suppliedAt time.Time) db.ListZoneFileStatusRow {
	return db.ListZoneFileStatusRow{
		SeedID:     seedID,
		NameDomain: pgtype.Text{String: domain, Valid: true},
		SuppliedAt: pgtype.Timestamptz{Time: suppliedAt, Valid: true},
	}
}

// staleZones flags only zones aged past two re-supply intervals, in the fixtures'
// own "N re-supply intervals" form, ordered by zone.
func TestStaleZonesPastTwoIntervals(t *testing.T) {
	now := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	cadence := int64(7 * 24 * 3600) // a 7-day re-supply interval
	day := 24 * time.Hour
	rows := []db.ListZoneFileStatusRow{
		staleRow(1, "fresh.acmecorp.io", now.Add(-3*day)),     // < 1 interval — fresh
		staleRow(2, "internal.acmecorp.io", now.Add(-20*day)), // ~2.85 intervals — stale
		staleRow(3, "ancient.acmecorp.io", now.Add(-30*day)),  // ~4.28 intervals — stale
	}
	got := staleZones(rows, cadence, now)
	if len(got) != 2 {
		t.Fatalf("want 2 stale zones, got %d (%v)", len(got), got)
	}
	// Ordered by zone.
	if got[0].Zone != "ancient.acmecorp.io" || got[1].Zone != "internal.acmecorp.io" {
		t.Fatalf("stale zones must be ordered by zone, got %q then %q", got[0].Zone, got[1].Zone)
	}
	if got[1].Age != "2 re-supply intervals" {
		t.Fatalf("age in the fixtures' form: want %q, got %q", "2 re-supply intervals", got[1].Age)
	}
	if got[0].Age != "4 re-supply intervals" {
		t.Fatalf("age floors the interval count: want %q, got %q", "4 re-supply intervals", got[0].Age)
	}
}

// A zone with no supplied file (or no domain) degrades to no callout — the design's
// empty pattern, never a fabricated zero.
func TestStaleZonesSkipsUnsupplied(t *testing.T) {
	now := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	cadence := int64(7 * 24 * 3600)
	rows := []db.ListZoneFileStatusRow{
		{SeedID: 1, NameDomain: pgtype.Text{String: "never.acmecorp.io", Valid: true}}, // SuppliedAt invalid
		{SeedID: 2, SuppliedAt: pgtype.Timestamptz{Time: now.Add(-100 * 24 * time.Hour), Valid: true}}, // no domain
	}
	if got := staleZones(rows, cadence, now); len(got) != 0 {
		t.Fatalf("unsupplied/no-domain zones must not appear, got %v", got)
	}
	if got := staleZones(nil, 0, now); got != nil {
		t.Fatalf("no re-supply interval declared: want nil, got %v", got)
	}
}
