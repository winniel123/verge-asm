package scan

import (
	"encoding/json"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/tlsacceptance"
)

// The tls-acceptance Scan fans out one job per Vantage over the Services reached
// from that Vantage — the open `Service` population — with NO port list consulted.
// Every reached Service here still passes the re-applied Custody gate: the operator
// scopes (operatorEstate) cover both addresses and each vantage's derived class admits
// them, so the enumeration is unchanged for a population still in scope.
func TestBuildTLSAcceptanceJobsOnePerVantageOverReachedServices(t *testing.T) {
	estate := operatorEstate()
	vantages := []Vantage{
		internetVantage(1, "internet"),
		internalVantage(2, "internal"),
	}
	services := []ReachedService{
		{VantageID: 1, Address: "93.184.216.10", Port: 443}, // operator, globally reachable
		{VantageID: 1, Address: "93.184.216.10", Port: 8443},
		{VantageID: 2, Address: "10.0.0.5", Port: 443}, // operator, private — internal vantage
	}
	jobs := BuildTLSAcceptanceJobs(9, estate, services, vantages)
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

// A reached Service whose authorising Custody condition has since been withdrawn is
// re-gated OUT and produces NO job (ADR-0079, #742) — the stale-population back door
// the connect-time gate would otherwise be bypassed through. Two withdrawals: the
// covering address scope removed (address now third-party), and the vantage's derived
// class flipped to `internet` over a still-in-scope private address (denotation).
func TestBuildTLSAcceptanceJobsReGatesWithdrawnService(t *testing.T) {
	// (1) Covering address scope withdrawn: an empty estate covers nothing, so a
	// once-reached address is now third-party and MayProbe refuses it.
	empty := custody.Estate{}
	services := []ReachedService{{VantageID: 1, Address: "93.184.216.10", Port: 443}}
	if jobs := BuildTLSAcceptanceJobs(1, empty, services, []Vantage{internetVantage(1, "internet")}); jobs != nil {
		t.Errorf("a reached Service whose address scope was withdrawn must yield no job, got %d", len(jobs))
	}

	// (2) Vantage class flipped to `internet`: the private address stays operator by
	// address scope but ADR-0079 bars a non-globally-reachable address from an
	// internet-class vantage, so it is re-gated out.
	estate := operatorEstate()
	priv := []ReachedService{{VantageID: 1, Address: "10.0.0.5", Port: 443}}
	if jobs := BuildTLSAcceptanceJobs(1, estate, priv, []Vantage{internetVantage(1, "internet")}); jobs != nil {
		t.Errorf("a private address must not be enumerated from an internet-class vantage, got %d jobs", len(jobs))
	}
	// Sanity: from an internal-class vantage the same in-scope address still passes.
	if jobs := BuildTLSAcceptanceJobs(1, estate, priv, []Vantage{internalVantage(1, "internal")}); len(jobs) != 1 {
		t.Errorf("an in-scope private address must still enumerate from an internal-class vantage, got %d jobs", len(jobs))
	}
}

// A Vantage with no reached Service, and empty inputs, each yield no jobs — a
// legible empty scope, not an error.
func TestBuildTLSAcceptanceJobsEmptyIsLegible(t *testing.T) {
	estate := operatorEstate()
	vantages := []Vantage{internetVantage(1, "internet")}
	if jobs := BuildTLSAcceptanceJobs(1, estate, nil, vantages); jobs != nil {
		t.Errorf("no reached Services should yield no jobs, got %d", len(jobs))
	}
	if jobs := BuildTLSAcceptanceJobs(1, estate, []ReachedService{{VantageID: 1, Address: "93.184.216.10", Port: 443}}, nil); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
	// A reached Service at an unconfigured Vantage is dropped, not enqueued.
	if jobs := BuildTLSAcceptanceJobs(1, estate, []ReachedService{{VantageID: 99, Address: "93.184.216.10", Port: 443}}, vantages); jobs != nil {
		t.Errorf("a Service at an unconfigured vantage must be dropped, got %d jobs", len(jobs))
	}
}

// The candidate set — the full declared TLS offer — travels in the job spec and is
// recorded on the Batch by content, never a library default (ADR-0025). It is NOT a
// port list: the scope carries Services with their own ports.
func TestTLSAcceptanceJobRecordsCandidateSetByContent(t *testing.T) {
	j := BuildTLSAcceptanceJobs(1, operatorEstate(),
		[]ReachedService{{VantageID: 1, Address: "93.184.216.10", Port: 443}},
		[]Vantage{internetVantage(1, "internet")},
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
