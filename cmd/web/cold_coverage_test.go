package main

import (
	"net/netip"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

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

func TestWalkedAddressesDropsNameHosts(t *testing.T) {
	rows := []db.ListCurrentServiceSubjectsRow{
		{SubjectKey: "203.0.113.44:22"},
		{SubjectKey: "203.0.113.9:443/tcp"},
		{SubjectKey: "mail.acmecorp.io:25"},
		{SubjectKey: "not-an-ip"},
		{SubjectKey: "198.51.100.7:80"},
	}
	got := walkedAddresses(rows)
	if len(got) != 3 {
		t.Fatalf("walkedAddresses: want 3 IP hosts, got %d (%v)", len(got), got)
	}
}

func TestCoveredInRangeDistinctAndBounded(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	walked := []walkedAddr{
		{Addr: mustAddr(t, "203.0.113.44")},
		{Addr: mustAddr(t, "203.0.113.44")},
		{Addr: mustAddr(t, "203.0.113.9")},
		{Addr: mustAddr(t, "198.51.100.7")},
		{Addr: mustAddr(t, "2001:db8::1")},
	}
	if got := coveredInRange(walked, p); got != 2 {
		t.Fatalf("coveredInRange: want 2 distinct in-range, got %d", got)
	}
	if got := coveredInRange(nil, p); got != 0 {
		t.Fatalf("coveredInRange(nil): want 0, got %d", got)
	}
}

func TestApertureMetersAddressCountedTotal(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	walked := []walkedAddr{
		{Addr: mustAddr(t, "203.0.113.10")},
		{Addr: mustAddr(t, "203.0.113.20")},
		{Addr: mustAddr(t, "203.0.113.30")},
	}
	m := apertureMeters(seeds, nil, walked, time.Now(), nil)
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

func TestApertureMetersAddressZeroNumerator(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	m := apertureMeters(seeds, nil, nil, time.Now(), nil)
	if m[0].Total == nil || m[0].Counted != "0" || m[0].Pct != 0 {
		t.Fatalf("empty walk: want 0/256 at 0%%, got counted=%q total=%v pct=%d", m[0].Counted, m[0].Total, m[0].Pct)
	}
}

func TestApertureMetersNameCensus(t *testing.T) {
	seeds := []db.ListSeedsRow{{Kind: "name", NameDomain: pgtype.Text{String: "acmecorp.io", Valid: true}}}
	m := apertureMeters(seeds, nil, nil, time.Now(), nil)
	if m[0].Total != nil {
		t.Fatalf("name scope must be a census (Total nil), got %v", m[0].Total)
	}
}

func TestOldestCurrentInRange(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	old := time.Date(2026, 8, 28, 6, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 8, 29, 6, 0, 0, 0, time.UTC)
	newest := time.Date(2026, 8, 30, 6, 0, 0, 0, time.UTC)
	walked := []walkedAddr{
		{Addr: mustAddr(t, "203.0.113.10"), ObservedAt: mid},
		{Addr: mustAddr(t, "203.0.113.20"), ObservedAt: old},
		{Addr: mustAddr(t, "203.0.113.30"), ObservedAt: newest},
		{Addr: mustAddr(t, "198.51.100.7"), ObservedAt: old.Add(-48 * time.Hour)},
		{Addr: mustAddr(t, "203.0.113.40")},
	}
	got, ok := oldestCurrentInRange(walked, p)
	if !ok {
		t.Fatalf("want an as-of, got ok=false")
	}
	if !got.Equal(old) {
		t.Fatalf("as-of = the oldest current in-range instant: want %s, got %s", old, got)
	}
	if _, ok := oldestCurrentInRange([]walkedAddr{{Addr: mustAddr(t, "203.0.113.40")}}, p); ok {
		t.Fatalf("a range with no real instant must read ok=false")
	}
	if _, ok := oldestCurrentInRange(nil, p); ok {
		t.Fatalf("oldestCurrentInRange(nil): want ok=false")
	}
}

func TestAddressMeterOldestCurrentAsOf(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	now := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	oldest := now.Add(-30 * time.Hour)
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	walked := []walkedAddr{
		{Addr: mustAddr(t, "203.0.113.10"), ObservedAt: now.Add(-1 * time.Hour)},
		{Addr: mustAddr(t, "203.0.113.20"), ObservedAt: oldest},
		{Addr: mustAddr(t, "203.0.113.30"), ObservedAt: now.Add(-3 * time.Hour)},
	}
	m := apertureMeters(seeds, nil, walked, now, nil)[0]
	if m.Total == nil || *m.Total != "256" || m.Counted != "3" {
		t.Fatalf("lagging meter is counted/total 3/256: got counted=%q total=%v", m.Counted, m.Total)
	}
	if m.AsOfISO != oldest.UTC().Format(time.RFC3339) {
		t.Fatalf("as-of ISO = the oldest current instant: want %q, got %q", oldest.UTC().Format(time.RFC3339), m.AsOfISO)
	}
	if m.AsOf != agoLabel(oldest, now) {
		t.Fatalf("as-of phrase = agoLabel(oldest, now)=%q, got %q", agoLabel(oldest, now), m.AsOf)
	}
}

func TestAddressMeterNoAsOfWhenNothingCurrent(t *testing.T) {
	p := mustPrefix(t, "203.0.113.0/24")
	seeds := []db.ListSeedsRow{{Kind: "address", AddressCidr: &p}}
	m := apertureMeters(seeds, nil, nil, time.Now(), nil)[0]
	if m.AsOf != "" || m.AsOfISO != "" {
		t.Fatalf("no current subject: as-of must be empty, got AsOf=%q AsOfISO=%q", m.AsOf, m.AsOfISO)
	}
}

func staleRow(seedID int64, domain string, suppliedAt time.Time) db.ListZoneFileStatusRow {
	return db.ListZoneFileStatusRow{
		SeedID:     seedID,
		NameDomain: pgtype.Text{String: domain, Valid: true},
		SuppliedAt: pgtype.Timestamptz{Time: suppliedAt, Valid: true},
	}
}

func TestStaleZonesPastTwoIntervals(t *testing.T) {
	now := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	cadence := int64(7 * 24 * 3600)
	day := 24 * time.Hour
	rows := []db.ListZoneFileStatusRow{
		staleRow(1, "fresh.acmecorp.io", now.Add(-3*day)),
		staleRow(2, "internal.acmecorp.io", now.Add(-20*day)),
		staleRow(3, "ancient.acmecorp.io", now.Add(-30*day)),
	}
	got := staleZones(rows, cadence, now)
	if len(got) != 2 {
		t.Fatalf("want 2 stale zones, got %d (%v)", len(got), got)
	}
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

func TestStaleZonesSkipsUnsupplied(t *testing.T) {
	now := time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC)
	cadence := int64(7 * 24 * 3600)
	rows := []db.ListZoneFileStatusRow{
		{SeedID: 1, NameDomain: pgtype.Text{String: "never.acmecorp.io", Valid: true}},
		{SeedID: 2, SuppliedAt: pgtype.Timestamptz{Time: now.Add(-100 * 24 * time.Hour), Valid: true}},
	}
	if got := staleZones(rows, cadence, now); len(got) != 0 {
		t.Fatalf("unsupplied/no-domain zones must not appear, got %v", got)
	}
	if got := staleZones(nil, 0, now); got != nil {
		t.Fatalf("no re-supply interval declared: want nil, got %v", got)
	}
}
