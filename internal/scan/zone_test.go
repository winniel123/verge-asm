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

func keys(m map[string]ZoneRecord) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
