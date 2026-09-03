package scan

import (
	"encoding/json"
	"iter"
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
)

func candidates(t *testing.T, raw ...string) iter.Seq[netip.Addr] {
	t.Helper()
	out := make([]netip.Addr, 0, len(raw))
	for _, s := range raw {
		out = append(out, netip.MustParseAddr(s))
	}
	return slices.Values(out)
}

func jobsOf(scanID int64, population iter.Seq[netip.Addr]) []EdgeFanoutJob {
	return slices.Collect(BuildEdgeFanoutJobs(scanID, population))
}

func TestBuildEdgeFanoutJobsCarriesTheCandidatesWithNoVantage(t *testing.T) {
	jobs := jobsOf(7, candidates(t, "93.184.216.34", "203.0.113.10"))
	if len(jobs) != 1 {
		t.Fatalf("jobs = %d, want 1", len(jobs))
	}
	if jobs[0].ScanID != 7 {
		t.Fatalf("ScanID = %d, want 7", jobs[0].ScanID)
	}
	want := []string{"93.184.216.34", "203.0.113.10"}
	if len(jobs[0].Addresses) != len(want) {
		t.Fatalf("addresses = %v, want %v", jobs[0].Addresses, want)
	}
	for i, w := range want {
		if jobs[0].Addresses[i] != w {
			t.Fatalf("addresses[%d] = %q, want %q", i, jobs[0].Addresses[i], w)
		}
	}
}

func TestBuildEdgeFanoutJobsEmptyIsLegible(t *testing.T) {
	// An install holding address scopes alone is not this case: its population is non-empty (#988).
	if jobs := jobsOf(1, candidates(t)); jobs != nil {
		t.Fatalf("jobs = %v, want none", jobs)
	}
	if jobs := jobsOf(1, slices.Values([]netip.Addr(nil))); jobs != nil {
		t.Fatalf("jobs = %v, want none", jobs)
	}
}

func TestBuildEdgeFanoutJobsChunksAboveTheBound(t *testing.T) {
	var addrs []netip.Addr
	// Serial handshakes in one job would outlast the worker's probe deadline, so the job chunks.
	for i := range EdgeFanoutAddressesPerJob*2 + 3 {
		addrs = append(addrs, netip.AddrFrom4([4]byte{198, 51, 100, byte(i)}))
	}
	jobs := jobsOf(1, slices.Values(addrs))
	if len(jobs) != 3 {
		t.Fatalf("jobs = %d, want 3", len(jobs))
	}
	var flat []string
	for i, j := range jobs {
		if len(j.Addresses) > EdgeFanoutAddressesPerJob {
			t.Fatalf("chunk of %d addresses exceeds the bound %d", len(j.Addresses), EdgeFanoutAddressesPerJob)
		}
		if j.Chunk != i {
			t.Fatalf("jobs[%d].Chunk = %d, want %d — two jobs of one tick would share a Batch label", i, j.Chunk, i)
		}
		flat = append(flat, j.Addresses...)
	}
	if len(flat) != len(addrs) {
		t.Fatalf("chunks cover %d addresses, want %d", len(flat), len(addrs))
	}
	for i, a := range addrs {
		if flat[i] != a.String() {
			t.Fatalf("flattened[%d] = %q, want %q", i, flat[i], a)
		}
	}
}

func TestEdgeFanoutJobSpecDecodesAsTheLeafScope(t *testing.T) {
	j := jobsOf(3, candidates(t, "93.184.216.34"))[0]
	spec, err := j.JobSpec("scan:3:chunk:0")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if spec.Kind != edgefanout.Kind {
		t.Fatalf("kind = %q, want %q", spec.Kind, edgefanout.Kind)
	}
	if spec.Batch != "scan:3:chunk:0" {
		t.Fatalf("batch = %q", spec.Batch)
	}
	scope, err := edgefanout.DecodeScope(spec)
	if err != nil {
		t.Fatalf("DecodeScope: %v", err)
	}
	if len(scope.Addresses) != 1 || scope.Addresses[0] != "93.184.216.34" {
		t.Fatalf("scope addresses = %v", scope.Addresses)
	}
}

func TestEdgeFanoutAttemptedScopeRecordsTheAddresses(t *testing.T) {
	j := jobsOf(1, candidates(t, "203.0.113.10", "203.0.113.11"))[0]
	raw, err := j.AttemptedScope()
	if err != nil {
		t.Fatalf("AttemptedScope: %v", err)
	}
	var got struct {
		Addresses []string `json:"addresses"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Addresses) != 2 || got.Addresses[0] != "203.0.113.10" || got.Addresses[1] != "203.0.113.11" {
		t.Fatalf("recorded addresses = %v", got.Addresses)
	}
}

func TestEdgeFanoutOffersAreEmpty(t *testing.T) {
	// The threshold is a versioned parameter of the Custody derivation, never an offer on the probe.
	j := jobsOf(1, candidates(t, "203.0.113.10"))[0]
	raw, err := j.OffersJSON()
	if err != nil {
		t.Fatalf("OffersJSON: %v", err)
	}
	if string(raw) != "{}" {
		t.Fatalf("offers = %s, want {}", raw)
	}
}

func TestBuildEdgeFanoutJobsRendersUnmapped(t *testing.T) {
	j := jobsOf(1, candidates(t, "::ffff:93.184.216.34"))[0]
	if j.Addresses[0] != "93.184.216.34" {
		t.Fatalf("address = %q, want 93.184.216.34", j.Addresses[0])
	}
}
