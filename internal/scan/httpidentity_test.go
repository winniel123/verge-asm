package scan

import (
	"encoding/json"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/httpexchange"
)

// The http-identity Scan fans out one job per Vantage over the Services reached from
// that Vantage — the open `Service` population, rendered as nameless `Endpoint`s —
// with NO port list consulted. This is the dispatch child #686 omitted; without it the
// `http-identity` facet is never persisted and the four HTTP rules sit outside-domain.
func TestBuildHTTPIdentityJobsOnePerVantageOverReachedServices(t *testing.T) {
	vantages := []Vantage{
		{ID: 1, Name: "internet", Class: "internet"},
		{ID: 2, Name: "internal", Class: "internal"},
	}
	services := []ReachedService{
		{VantageID: 1, Address: "198.51.100.10", Port: 443},
		{VantageID: 1, Address: "198.51.100.10", Port: 80},
		{VantageID: 2, Address: "10.0.0.5", Port: 8080},
	}
	jobs := BuildHTTPIdentityJobs(9, services, vantages)
	if len(jobs) != 2 {
		t.Fatalf("got %d jobs, want one per vantage with reached Services (2)", len(jobs))
	}
	if jobs[0].VantageID != 1 || len(jobs[0].Targets) != 2 {
		t.Errorf("vantage 1 job = %+v, want its two reached Endpoints", jobs[0])
	}
	if jobs[1].VantageID != 2 || len(jobs[1].Targets) != 1 {
		t.Errorf("vantage 2 job = %+v, want its one reached Endpoint", jobs[1])
	}
	for _, j := range jobs {
		if j.Kind != httpexchange.Kind {
			t.Errorf("job kind = %q, want %q (the http-exchange leaf)", j.Kind, httpexchange.Kind)
		}
		for _, tgt := range j.Targets {
			if tgt.Name != "" {
				t.Errorf("reached Service renders the nameless Endpoint, got name %q", tgt.Name)
			}
		}
	}
}

// The scheme frames the single GET / — https on the implicit-TLS ports, http
// elsewhere — a framing of how the exchange is spoken, never a widening of the reached
// population.
func TestBuildHTTPIdentityJobsSchemeFramedByPort(t *testing.T) {
	services := []ReachedService{
		{VantageID: 1, Address: "198.51.100.10", Port: 443},
		{VantageID: 1, Address: "198.51.100.10", Port: 8443},
		{VantageID: 1, Address: "198.51.100.10", Port: 80},
		{VantageID: 1, Address: "198.51.100.10", Port: 8080},
	}
	j := BuildHTTPIdentityJobs(1, services, []Vantage{{ID: 1, Name: "internet"}})[0]
	want := map[uint16]string{443: "https", 8443: "https", 80: "http", 8080: "http"}
	for _, tgt := range j.Targets {
		if got := tgt.Scheme; got != want[tgt.Port] {
			t.Errorf("port %d scheme = %q, want %q", tgt.Port, got, want[tgt.Port])
		}
	}
}

// A Vantage with no reached Service, and empty inputs, each yield no jobs — a legible
// empty scope, not an error.
func TestBuildHTTPIdentityJobsEmptyIsLegible(t *testing.T) {
	vantages := []Vantage{{ID: 1, Name: "internet", Class: "internet"}}
	if jobs := BuildHTTPIdentityJobs(1, nil, vantages); jobs != nil {
		t.Errorf("no reached Services should yield no jobs, got %d", len(jobs))
	}
	if jobs := BuildHTTPIdentityJobs(1, []ReachedService{{VantageID: 1, Address: "198.51.100.1", Port: 80}}, nil); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
	// A reached Service at an unconfigured Vantage is dropped, not enqueued.
	if jobs := BuildHTTPIdentityJobs(1, []ReachedService{{VantageID: 99, Address: "198.51.100.1", Port: 80}}, vantages); jobs != nil {
		t.Errorf("a Service at an unconfigured vantage must be dropped, got %d jobs", len(jobs))
	}
}

// The declared parameter set — the §3.3 HTTP safety table — travels in the job spec
// and is recorded on the Batch by content, never a library default (ADR-0025). It is
// NOT a port list: the scope carries Endpoints with their own ports. The spec's Kind is
// the http-exchange leaf, so the prober's existing httpexchange case handles it.
func TestHTTPIdentityJobRecordsParamsByContent(t *testing.T) {
	j := BuildHTTPIdentityJobs(1,
		[]ReachedService{{VantageID: 1, Address: "198.51.100.10", Port: 443}},
		[]Vantage{{ID: 1, Name: "internet", Class: "internet"}},
	)[0]

	spec, err := j.JobSpec("job-1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != httpexchange.Kind {
		t.Errorf("spec kind = %q, want %q", spec.Kind, httpexchange.Kind)
	}
	scope, err := httpexchange.DecodeScope(spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(scope.Targets) != 1 || scope.Targets[0].Port != 443 || scope.Targets[0].Scheme != "https" {
		t.Errorf("targets carry their own ports/scheme, no port list: %+v", scope.Targets)
	}
	if scope.Params.Digest() != httpexchange.DefaultParams().Digest() {
		t.Errorf("scope params not recorded by content: %+v", scope.Params)
	}

	// Offers recorded on the Batch equal the declared parameter set.
	offers, err := j.OffersJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got httpexchange.Params
	if err := json.Unmarshal(offers, &got); err != nil {
		t.Fatal(err)
	}
	if got.Digest() != httpexchange.DefaultParams().Digest() {
		t.Errorf("recorded offers digest %s != declared %s", got.Digest(), httpexchange.DefaultParams().Digest())
	}
}

// A dead-lettered Batch records an empty scope — never the attempted Endpoints, which
// would manufacture HTTP-identity absences it never measured (v1 spec §4.1).
func TestEmptyHTTPIdentityScopeHasNoTargets(t *testing.T) {
	b, err := EmptyHTTPIdentityScope("internet")
	if err != nil {
		t.Fatal(err)
	}
	var rec httpIdentityScopeRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Targets) != 0 {
		t.Errorf("dead-lettered scope must be empty, got %v", rec.Targets)
	}
}
