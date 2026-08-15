package scan

import (
	"encoding/json"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

// The tls-acceptance Scan fans out one job per Vantage over the Services reached
// from that Vantage — the open `Service` population — with NO port list consulted.
func TestBuildTLSAcceptanceJobsOnePerVantageOverReachedServices(t *testing.T) {
	vantages := []Vantage{
		{ID: 1, Name: "internet", Class: "internet"},
		{ID: 2, Name: "internal", Class: "internal"},
	}
	services := []ReachedService{
		{VantageID: 1, Address: "198.51.100.10", Port: 443},
		{VantageID: 1, Address: "198.51.100.10", Port: 8443},
		{VantageID: 2, Address: "10.0.0.5", Port: 443},
	}
	jobs := BuildTLSAcceptanceJobs(9, services, vantages)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want one per vantage with reached Services (2)", len(jobs))
	}
	if jobs[0].VantageID != 1 || len(jobs[0].Services) != 2 {
		t.Errorf("vantage 1 job = %+v, want its two reached Services", jobs[0])
	}
	if jobs[1].VantageID != 2 || len(jobs[1].Services) != 1 {
		t.Errorf("vantage 2 job = %+v, want its one reached Service", jobs[1])
	}
	for _, j := range jobs {
		if j.Kind != tlsacceptance.Kind {
			t.Errorf("job kind = %q, want %q (its own leaf)", j.Kind, tlsacceptance.Kind)
		}
	}
}

// A Vantage with no reached Service, and empty inputs, each yield no jobs — a
// legible empty scope, not an error.
func TestBuildTLSAcceptanceJobsEmptyIsLegible(t *testing.T) {
	vantages := []Vantage{{ID: 1, Name: "internet", Class: "internet"}}
	if jobs := BuildTLSAcceptanceJobs(1, nil, vantages); jobs != nil {
		t.Errorf("no reached Services should yield no jobs, got %d", len(jobs))
	}
	if jobs := BuildTLSAcceptanceJobs(1, []ReachedService{{VantageID: 1, Address: "198.51.100.1", Port: 443}}, nil); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
	// A reached Service at an unconfigured Vantage is dropped, not enqueued.
	if jobs := BuildTLSAcceptanceJobs(1, []ReachedService{{VantageID: 99, Address: "198.51.100.1", Port: 443}}, vantages); jobs != nil {
		t.Errorf("a Service at an unconfigured vantage must be dropped, got %d jobs", len(jobs))
	}
}

// The candidate set — the full declared TLS offer — travels in the job spec and is
// recorded on the Batch by content, never a library default (ADR-0025). It is NOT a
// port list: the scope carries Services with their own ports.
func TestTLSAcceptanceJobRecordsCandidateSetByContent(t *testing.T) {
	j := BuildTLSAcceptanceJobs(1,
		[]ReachedService{{VantageID: 1, Address: "198.51.100.10", Port: 443}},
		[]Vantage{{ID: 1, Name: "internet", Class: "internet"}},
	)[0]

	spec, err := j.JobSpec("job-1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != tlsacceptance.Kind {
		t.Errorf("spec kind = %q, want %q", spec.Kind, tlsacceptance.Kind)
	}
	var scope tlsacceptance.Scope
	if err := json.Unmarshal(spec.Scope, &scope); err != nil {
		t.Fatal(err)
	}
	want := tlsacceptance.DefaultCandidateSet()
	if len(scope.Candidates.Versions) != len(want.Versions) || len(scope.Candidates.Ciphers) != len(want.Ciphers) {
		t.Errorf("candidate set not recorded by content: %+v", scope.Candidates)
	}
	if len(scope.Services) != 1 || scope.Services[0].Port != 443 {
		t.Errorf("services carry their own ports, no port list: %+v", scope.Services)
	}

	// Offers recorded on the Batch equal the declared candidate set.
	offers, err := j.OffersJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got tlsacceptance.CandidateSet
	if err := json.Unmarshal(offers, &got); err != nil {
		t.Fatal(err)
	}
	if got.Digest() != want.Digest() {
		t.Errorf("recorded offers digest %s != declared %s", got.Digest(), want.Digest())
	}
}

// A dead-lettered Batch records an empty scope — never the attempted Services,
// which would manufacture acceptance absences it never measured (v1 spec §4.1).
func TestEmptyTLSAcceptanceScopeHasNoServices(t *testing.T) {
	b, err := EmptyTLSAcceptanceScope("internet")
	if err != nil {
		t.Fatal(err)
	}
	var rec tlsAcceptanceScopeRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Services) != 0 {
		t.Errorf("dead-lettered scope must be empty, got %v", rec.Services)
	}
}
