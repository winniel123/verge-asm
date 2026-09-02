package scan

import (
	"encoding/json"
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

func addr(s string) netip.Addr { return netip.MustParseAddr(s) }

// hotJobs collects the streamed hot fan-out into a slice for assertions. The
// builder yields one job per (Vantage, admitted address); collecting an empty
// sequence returns nil, so a nil check still reads as "no jobs".
func hotJobs(scanID int64, estate custody.Estate, addrs []netip.Addr, vantages []Vantage, core vergecore.List) []HotJob {
	return slices.Collect(BuildHotJobs(scanID, estate, slices.Values(addrs), vantages, core))
}

func operatorEstate() custody.Estate {
	return custody.Estate{
		AddressScopes: []netip.Prefix{
			netip.MustParsePrefix("93.184.216.0/24"), // operator, globally reachable
			netip.MustParsePrefix("10.0.0.0/8"),      // operator, non-globally-reachable
		},
	}
}

// internetVantage / internalVantage build a Vantage whose class the hot/cold Scans now
// DERIVE per batch from its presented dialled address against operatorEstate's scopes
// (#709), rather than a stored column: 203.0.113.1 is covered by no scope → `internet`,
// 10.0.0.9 is inside 10.0.0.0/8 → `internal`.
func internetVantage(id int64, name string) Vantage {
	return Vantage{ID: id, Name: name, Dialled: "203.0.113.1"}
}

func internalVantage(id int64, name string) Vantage {
	return Vantage{ID: id, Name: name, Dialled: "10.0.0.9"}
}

// The whole of the gate: a third-party address is never handed to a prober, on
// no port, from no Vantage. This is the load-bearing safety assertion (ADR-0019).
func TestBuildHotJobsNeverProbesThirdParty(t *testing.T) {
	estate := operatorEstate()
	third := "8.8.4.4" // covered by no Seed — third-party
	addrs := []netip.Addr{addr("93.184.216.10"), addr(third)}
	vantages := []Vantage{internetVantage(1, "internet-a")}

	jobs := hotJobs(7, estate, addrs, vantages, vergecore.Default())
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1 (one vantage with an admitted address)", len(jobs))
	}
	for _, j := range jobs {
		for _, a := range j.Addresses {
			if a == third {
				t.Fatalf("third-party address %s appeared in a hot job — the Custody gate did not hold", third)
			}
		}
		if len(j.Addresses) != 1 || j.Addresses[0] != "93.184.216.10" {
			t.Errorf("admitted addresses = %v, want only the operator address", j.Addresses)
		}
	}
}

// A non-globally-reachable operator address is barred from an `internet`-class
// Vantage (denotation, ADR-0079) and admitted from an `internal`-class one.
func TestBuildHotJobsDenotationGate(t *testing.T) {
	estate := operatorEstate()
	priv := "10.0.0.5" // operator by address scope, non-globally-reachable
	addrs := []netip.Addr{addr(priv)}

	internet := []Vantage{internetVantage(1, "net")}
	if jobs := hotJobs(1, estate, addrs, internet, vergecore.Default()); len(jobs) != 0 {
		t.Errorf("a private address must not be probed from an internet-class vantage, got %d jobs", len(jobs))
	}

	internal := []Vantage{internalVantage(2, "lan")}
	jobs := hotJobs(1, estate, addrs, internal, vergecore.Default())
	if len(jobs) != 1 || len(jobs[0].Addresses) != 1 {
		t.Fatalf("a private address must be probed from an internal-class vantage, got %d jobs", len(jobs))
	}
}

// The hot job carries the verge-core TCP set to probe and the UDP set to record,
// and dispatches the connect-outcome leaf.
func TestBuildHotJobsCarriesVergeCore(t *testing.T) {
	estate := operatorEstate()
	addrs := []netip.Addr{addr("93.184.216.10")}
	vantages := []Vantage{internetVantage(1, "net")}
	core := vergecore.Default()

	jobs := hotJobs(1, estate, addrs, vantages, core)
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.Kind != connectoutcome.Kind {
		t.Errorf("job kind = %q, want %q", j.Kind, connectoutcome.Kind)
	}
	if len(j.TCPPorts) != core.Count().TCP {
		t.Errorf("job carries %d TCP ports, want %d (the whole probed set)", len(j.TCPPorts), core.Count().TCP)
	}
	if len(j.UDPPorts) != core.Count().UDP {
		t.Errorf("job carries %d UDP ports, want %d (recorded, never probed)", len(j.UDPPorts), core.Count().UDP)
	}

	// The scope on the wire is a connect-outcome scope carrying the profile.
	spec, err := j.JobSpec("job-1")
	if err != nil {
		t.Fatal(err)
	}
	var sc connectoutcome.Scope
	if err := json.Unmarshal(spec.Scope, &sc); err != nil {
		t.Fatal(err)
	}
	if sc.Profile.PerHostConnPerSec != 50 || sc.Profile.PerVantagePacketsPerSec != 200 {
		t.Errorf("safety profile not recorded by content: %+v", sc.Profile)
	}
	if len(sc.UDPPorts) != core.Count().UDP {
		t.Errorf("UDP ports must travel in scope to be recorded, got %d", len(sc.UDPPorts))
	}
}

// The measured half of #1106. ADR-0005 justified keeping the per-host pacer in
// prober memory on an intra-job argument, and that argument holds only while one
// host sits in one job. The fan-out is one job per (Vantage, admitted address),
// so at two vantages one host sits in two jobs, each carrying its own copy of the
// 50 conn/s per-host ceiling. Two probers can then be pointed at that one host.
func TestBuildHotJobsPutOneHostInOneJobPerVantage(t *testing.T) {
	estate := operatorEstate()
	host := "93.184.216.10"
	addrs := []netip.Addr{addr(host)}
	vantages := []Vantage{
		internetVantage(1, "internet-a"),
		{ID: 2, Name: "internet-b", Dialled: "198.51.100.7"},
	}

	jobs := hotJobs(1, estate, addrs, vantages, vergecore.Default())
	if len(jobs) != len(vantages) {
		t.Fatalf("got %d jobs for one address at %d vantages, want %d", len(jobs), len(vantages), len(vantages))
	}

	seen := map[int64]bool{}
	for _, j := range jobs {
		if len(j.Addresses) != 1 || j.Addresses[0] != host {
			t.Fatalf("job from vantage %d carries %v, want only %s", j.VantageID, j.Addresses, host)
		}
		if seen[j.VantageID] {
			t.Fatalf("vantage %d produced two jobs for one address", j.VantageID)
		}
		seen[j.VantageID] = true
		if j.Profile.PerHostConnPerSec != 50 {
			t.Errorf("per-host ceiling = %d, want 50 in every job", j.Profile.PerHostConnPerSec)
		}
	}
	if len(seen) != len(vantages) {
		t.Fatalf("one host reached %d vantages, want %d", len(seen), len(vantages))
	}
}

// Empty inputs are legible: no addresses, or no vantages, yields no jobs.
func TestBuildHotJobsEmptyIsLegible(t *testing.T) {
	estate := operatorEstate()
	v := []Vantage{internetVantage(1, "net")}
	if jobs := hotJobs(1, estate, nil, v, vergecore.Default()); jobs != nil {
		t.Errorf("no addresses should yield no jobs, got %d", len(jobs))
	}
	if jobs := hotJobs(1, estate, []netip.Addr{addr("93.184.216.10")}, nil, vergecore.Default()); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
}

// A dead-lettered hot Batch records an empty scope, never the attempted one.
func TestEmptyHotScope(t *testing.T) {
	b, err := EmptyHotScope("net")
	if err != nil {
		t.Fatal(err)
	}
	var rec hotScopeRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	if len(rec.Addresses) != 0 {
		t.Errorf("dead-lettered scope must be empty, got %v", rec.Addresses)
	}
}
