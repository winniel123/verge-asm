package scan

import (
	"encoding/json"
	"iter"
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
)

// candidates is a fixed population as the lazy sequence the builder now takes.
func candidates(t *testing.T, raw ...string) iter.Seq[netip.Addr] {
	t.Helper()
	out := make([]netip.Addr, 0, len(raw))
	for _, s := range raw {
		out = append(out, netip.MustParseAddr(s))
	}
	return slices.Values(out)
}

// jobsOf collects the streamed jobs so a test can assert over the whole tick. The
// dispatcher never collects — it enqueues in chunks (queue.streamEnqueue).
func jobsOf(scanID int64, population iter.Seq[netip.Addr]) []EdgeFanoutJob {
	return slices.Collect(BuildEdgeFanoutJobs(scanID, population))
}

// The fan-out is over addresses alone. There is no vantage dimension and no port list:
// a default certificate is not a function of vantage, and the edge is measured on
// 443/tcp alone.
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

// An instance with neither a custody extension nor a declared address scope dispatches an
// empty scope and enqueues no job. The empty scope is a legible state, not an error. Since
// #988 an install holding address scopes alone is NOT this case — its population is
// non-empty (custody.TestEdgeFanoutPopulationNonEmptyWithNoExtension).
func TestBuildEdgeFanoutJobsEmptyIsLegible(t *testing.T) {
	if jobs := jobsOf(1, candidates(t)); jobs != nil {
		t.Fatalf("jobs = %v, want none", jobs)
	}
	if jobs := jobsOf(1, slices.Values([]netip.Addr(nil))); jobs != nil {
		t.Fatalf("jobs = %v, want none", jobs)
	}
}

// A population above the per-job bound splits into chunks rather than riding one job
// whose serial handshakes would outlast the worker's probe deadline. Every address lands
// in exactly one chunk, in order, no chunk exceeds the bound, and Chunk numbers them from
// zero so the Batch labels of one tick are distinct.
func TestBuildEdgeFanoutJobsChunksAboveTheBound(t *testing.T) {
	var addrs []netip.Addr
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

// The JobSpec the dispatcher writes is exactly the scope the leaf decodes: the wire
// shape is shared by construction, so a dispatched job cannot fail to be read.
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

// The recorded scope is the by-content record of what the job set out to measure, under
// the `addresses` key the recording-side scope gate reads.
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

// The Scan declares no measurement parameter an operator chooses: the fan-out threshold
// is a versioned parameter of the `Custody` derivation, never an offer on the probe.
func TestEdgeFanoutOffersAreEmpty(t *testing.T) {
	j := jobsOf(1, candidates(t, "203.0.113.10"))[0]
	raw, err := j.OffersJSON()
	if err != nil {
		t.Fatalf("OffersJSON: %v", err)
	}
	if string(raw) != "{}" {
		t.Fatalf("offers = %s, want {}", raw)
	}
}

// An IPv4-mapped candidate is rendered in its unmapped netip form, so the scope the job
// carries and the address the leaf reports are the same spelling.
func TestBuildEdgeFanoutJobsRendersUnmapped(t *testing.T) {
	j := jobsOf(1, candidates(t, "::ffff:93.184.216.34"))[0]
	if j.Addresses[0] != "93.184.216.34" {
		t.Fatalf("address = %q, want 93.184.216.34", j.Addresses[0])
	}
}
