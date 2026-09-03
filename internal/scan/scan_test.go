package scan

import (
	"encoding/json"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

func TestBuildDNSJobsOnePerVantageEveryName(t *testing.T) {
	names := []string{"example.com", "example.net"}
	vantages := []Vantage{
		{ID: 1, Name: "local", Resolver: "10.0.0.1:53"},
		{ID: 2, Name: "prober-a", Resolver: "9.9.9.9:53"},
	}
	jobs := BuildDNSJobs(7, names, vantages)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want one per vantage (2)", len(jobs))
	}
	for i, j := range jobs {
		if j.ScanID != 7 {
			t.Errorf("job %d scan id = %d, want 7", i, j.ScanID)
		}
		if len(j.Names) != 2 {
			t.Errorf("job %d covers %d names, want every name (2)", i, len(j.Names))
		}
		if j.Kind != resolutionwalk.Kind {
			t.Errorf("job %d kind = %q, want %q", i, j.Kind, resolutionwalk.Kind)
		}
	}
}

func TestBuildDNSJobsEmptyScopeIsLegible(t *testing.T) {
	if jobs := BuildDNSJobs(1, nil, []Vantage{{ID: 1, Name: "local"}}); jobs != nil {
		t.Errorf("no names should yield no jobs, got %d", len(jobs))
	}
	if jobs := BuildDNSJobs(1, []string{"example.com"}, nil); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
}

func TestJobSpecRecordsOffersByContent(t *testing.T) {
	j := BuildDNSJobs(1, []string{"example.com"}, []Vantage{{ID: 1, Name: "local", Resolver: "10.0.0.1:53"}})[0].
		WithResolver("10.0.0.1:53")
	spec, err := j.JobSpec("job-1")
	if err != nil {
		t.Fatal(err)
	}
	var scope resolutionwalk.Scope
	if err := json.Unmarshal(spec.Scope, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Resolver != "10.0.0.1:53" {
		t.Errorf("resolver = %q, want the vantage's 10.0.0.1:53", scope.Resolver)
	}
	want := resolutionwalk.DefaultOffers()
	if len(scope.Offers.Qtypes) != len(want.Qtypes) || scope.Offers.EDNS.UDPBufSize != 1232 {
		t.Errorf("offers not recorded by content: %+v", scope.Offers)
	}
}

func TestAttemptedScopeControlPopulationWidensWithAdmittedNames(t *testing.T) {
	// The Seed apex's parent is a TLD outside every Seed, so the probing gate drops it (ADR-0107).
	j := BuildDNSJobs(1,
		[]string{"example.com", "vpn.example.com", "a.b.example.com"},
		[]Vantage{{ID: 1, Name: "local", Resolver: "10.0.0.1:53"}},
	)[0].WithSeeds([]string{"example.com"})

	raw, err := j.AttemptedScope()
	if err != nil {
		t.Fatal(err)
	}
	var rec scopeRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"example.com": true, "b.example.com": true}
	if len(rec.ControlProbePopulation) != len(want) {
		t.Fatalf("control-probe population = %v, want %v", rec.ControlProbePopulation, want)
	}
	for _, p := range rec.ControlProbePopulation {
		if !want[p] {
			t.Errorf("control-probe population has %q, which is not a parent inside the Seed", p)
		}
	}
	if len(rec.Names) != 3 {
		t.Errorf("recorded names = %v, want all three resolved names", rec.Names)
	}
}

func TestEmptyScopeHasNoNames(t *testing.T) {
	b, err := EmptyScope("local")
	if err != nil {
		t.Fatal(err)
	}
	var rec scopeRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Names) != 0 {
		t.Errorf("dead-lettered scope must be empty, got %v", rec.Names)
	}
}
