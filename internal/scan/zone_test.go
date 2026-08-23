package scan

import (
	"encoding/json"
	"testing"
	"time"
)

const sampleZone = `$ORIGIN example.com.
$TTL 3600
@       IN SOA ns1.example.com. hostmaster.example.com. (
                2026010101 ; serial
                7200       ; refresh
                3600       ; retry
                1209600    ; expire
                3600 )     ; minimum
@       IN NS   ns1.example.com.
@       IN A    203.0.113.10
        IN A    203.0.113.11
www     IN CNAME example.com.
mail    3600 IN A 203.0.113.20
api.example.com. IN AAAA 2001:db8::1
`

func TestRestateZoneStampsSupplyInstantNotReadTime(t *testing.T) {
	supply := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	// The worker reads the file much later; the observations must still be
	// stamped at the supply instant, never at this later read.
	zf := ZoneFile{Domain: "example.com", SuppliedAt: supply, Content: sampleZone}

	recs := RestateZone(zf)
	if len(recs) == 0 {
		t.Fatal("expected observations from the zone file, got none")
	}
	for _, r := range recs {
		if !r.ObservedAt.Equal(supply) {
			t.Errorf("record %s/%s stamped at %s, want the supply instant %s",
				r.Name, r.Qtype, r.ObservedAt, supply)
		}
	}
}

func TestRestateZoneGroupsRRsetsAndResolvesNames(t *testing.T) {
	supply := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	recs := RestateZone(ZoneFile{Domain: "example.com", SuppliedAt: supply, Content: sampleZone})

	byKey := map[string]ZoneRecord{}
	for _, r := range recs {
		byKey[r.Name+"/"+r.Qtype] = r
	}

	// The apex A RRset groups both address records into one dns-record
	// observation, keyed on (name, qtype) — the timeline shape.
	apexA, ok := byKey["example.com/A"]
	if !ok {
		t.Fatalf("apex A record missing; got %v", keys(byKey))
	}
	var v struct {
		RRs []string `json:"rrs"`
	}
	if err := json.Unmarshal(apexA.Data, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.RRs) != 2 {
		t.Errorf("apex A rrset = %v, want both 203.0.113.10 and .11", v.RRs)
	}

	// A blank-owner line inherits the previous owner (the second apex A above).
	// A relative name is qualified by $ORIGIN; an absolute name is kept as-is.
	if _, ok := byKey["www.example.com/CNAME"]; !ok {
		t.Errorf("relative owner not qualified by origin; got %v", keys(byKey))
	}
	if _, ok := byKey["mail.example.com/A"]; !ok {
		t.Errorf("owner with an explicit TTL prefix not parsed; got %v", keys(byKey))
	}
	if _, ok := byKey["api.example.com/AAAA"]; !ok {
		t.Errorf("absolute owner name not kept; got %v", keys(byKey))
	}
	// The multi-line SOA is one record, not five stray lines.
	if _, ok := byKey["example.com/SOA"]; !ok {
		t.Errorf("multi-line SOA not joined into one record; got %v", keys(byKey))
	}
}

func TestBuildZoneJobsOnePerFileEmptyIsLegible(t *testing.T) {
	if jobs := BuildZoneJobs(5, nil); jobs != nil {
		t.Errorf("no files should yield no jobs, got %d", len(jobs))
	}
	files := []ZoneFile{
		{SeedID: 1, Domain: "example.com", SuppliedAt: time.Now(), Content: sampleZone},
		{SeedID: 2, Domain: "example.net", SuppliedAt: time.Now(), Content: ""},
	}
	jobs := BuildZoneJobs(5, files)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want one per supplied file (2)", len(jobs))
	}
	if jobs[0].ScanID != 5 || jobs[0].SeedID != 1 || jobs[0].Domain != "example.com" {
		t.Errorf("job mis-built: %+v", jobs[0])
	}
}

func TestZoneJobSpecRoundTripsThroughWire(t *testing.T) {
	supply := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	j := BuildZoneJobs(1, []ZoneFile{{SeedID: 1, Domain: "example.com", SuppliedAt: supply, Content: sampleZone}})[0]
	spec, err := j.JobSpec("batch-1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != ZoneKind {
		t.Errorf("spec kind = %q, want %q", spec.Kind, ZoneKind)
	}
	zf, err := ZoneScopeFromSpec(spec.Scope)
	if err != nil {
		t.Fatal(err)
	}
	if !zf.SuppliedAt.Equal(supply) {
		t.Errorf("supply instant lost on the wire: %s != %s", zf.SuppliedAt, supply)
	}
	if zf.Content != sampleZone || zf.Domain != "example.com" {
		t.Errorf("zone scope round-trip corrupted content/domain")
	}
}

func TestZoneAgingCurrentFileCountsDownToTheGap(t *testing.T) {
	supply := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	interval := 30 * 24 * time.Hour // monthly
	// 23 days after supply: still current, seven days from ageing into a gap.
	now := supply.Add(23 * 24 * time.Hour)

	a := ZoneAgingAt(supply, now, interval)
	if !a.Supplied {
		t.Fatal("a dated supply should report Supplied")
	}
	if a.Stale {
		t.Fatalf("a file 23d into a 30d interval is not stale yet: %+v", a)
	}
	if a.Days != 7 {
		t.Errorf("Days = %d, want 7 (ages into a gap in 7d)", a.Days)
	}
}

func TestZoneAgingPastTheIntervalIsAGap(t *testing.T) {
	supply := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	interval := 30 * 24 * time.Hour
	// Five days past the interval: the file has aged into a coverage gap.
	now := supply.Add(35 * 24 * time.Hour)

	a := ZoneAgingAt(supply, now, interval)
	if !a.Stale {
		t.Fatalf("a file 35d into a 30d interval must be stale (a gap): %+v", a)
	}
	if a.Days != 5 {
		t.Errorf("Days = %d, want 5 (aged into a gap 5d ago)", a.Days)
	}
}

func TestZoneAgingBoundaryAndNoSupply(t *testing.T) {
	supply := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	interval := 30 * 24 * time.Hour
	// Exactly at the gap instant: stale, zero days in.
	if a := ZoneAgingAt(supply, supply.Add(interval), interval); !a.Stale || a.Days != 0 {
		t.Errorf("at the gap instant, want stale with 0 days; got %+v", a)
	}
	// A current file just under a day from the gap rounds up to 1d, never 0d.
	almost := supply.Add(interval - 12*time.Hour)
	if a := ZoneAgingAt(supply, almost, interval); a.Stale || a.Days != 1 {
		t.Errorf("half a day from the gap should read 1d and current; got %+v", a)
	}
	// No supply: nothing to stale.
	if a := ZoneAgingAt(time.Time{}, supply, interval); a.Supplied || a.Stale {
		t.Errorf("an unsupplied scope has nothing to age; got %+v", a)
	}
	// No cadence: current, no countdown.
	if a := ZoneAgingAt(supply, supply.Add(1000*24*time.Hour), 0); !a.Supplied || a.Stale {
		t.Errorf("with no interval a file cannot age into a gap; got %+v", a)
	}
}

func keys(m map[string]ZoneRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
