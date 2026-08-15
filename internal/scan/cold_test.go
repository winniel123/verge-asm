package scan

import (
	"encoding/json"
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
)

// coldEstate is the operator's declared estate for the cold tests: one globally
// reachable block and one non-globally-reachable block, both operator-owned so
// the Custody gate admits them and the opt-in gate is the thing under test.
func coldEstate() custody.Estate {
	return custody.Estate{
		AddressScopes: []netip.Prefix{
			netip.MustParsePrefix("93.184.216.0/24"),
			netip.MustParsePrefix("10.0.0.0/8"),
		},
	}
}

// The load-bearing invariant: with no Seed opted in, the cold Scan fans out
// nothing at all. An empty scope list is the shipped state (ADR-0044), so the
// full-range tier never fires unasked — not at onboarding, not on any tick.
func TestBuildColdJobsEmptyScopeNeverFires(t *testing.T) {
	estate := coldEstate()
	addrs := []netip.Addr{addr("93.184.216.10"), addr("10.0.0.5")}
	vantages := []Vantage{{ID: 1, Name: "internet-a", Class: string(custody.ClassInternet)}}

	if jobs := BuildColdJobs(1, estate, addrs, vantages, ColdScope{}); jobs != nil {
		t.Fatalf("an empty opt-in scope must produce no cold jobs, got %d", len(jobs))
	}
}

// Opting in is per-Seed scope: only addresses inside an opted-in scope are
// probed, and an operator-owned, Custody-admitted address that no Seed opted in
// stays out of the sweep entirely.
func TestBuildColdJobsOptInIsPerSeedScope(t *testing.T) {
	estate := coldEstate()
	optedIn := "93.184.216.10"
	notOptedIn := "93.184.216.11" // same block, operator-owned, but no Seed opted it in
	addrs := []netip.Addr{addr(optedIn), addr(notOptedIn)}
	vantages := []Vantage{{ID: 1, Name: "internet-a", Class: string(custody.ClassInternet)}}

	// Only one address is opted in, by explicit address membership.
	scope := ColdScope{Addresses: map[netip.Addr]bool{addr(optedIn): true}}
	jobs := BuildColdJobs(1, estate, addrs, vantages, scope)
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1 (one vantage with an opted-in address)", len(jobs))
	}
	if len(jobs[0].Addresses) != 1 || jobs[0].Addresses[0] != optedIn {
		t.Fatalf("cold job probed %v, want only the opted-in address %s", jobs[0].Addresses, optedIn)
	}
}

// An opted-in scope may be an address-scope Seed's CIDR: every address inside it
// is in the sweep, exactly as the Seed enumerates it.
func TestBuildColdJobsAddressScopeMembership(t *testing.T) {
	estate := coldEstate()
	addrs := []netip.Addr{addr("93.184.216.10"), addr("93.184.216.42")}
	vantages := []Vantage{{ID: 1, Name: "internet-a", Class: string(custody.ClassInternet)}}

	scope := ColdScope{AddressPrefixes: []netip.Prefix{netip.MustParsePrefix("93.184.216.0/24")}}
	jobs := BuildColdJobs(1, estate, addrs, vantages, scope)
	if len(jobs) != 1 || len(jobs[0].Addresses) != 2 {
		t.Fatalf("an opted-in CIDR must cover every address inside it, got %+v", jobs)
	}
}

// The Custody gate is reused unchanged: an opted-in address that is non-globally-
// reachable is still barred from an internet-class Vantage (ADR-0019/ADR-0079).
// The opt-in gate widens the tier; it can never move an address past Custody.
func TestBuildColdJobsStillCustodyGated(t *testing.T) {
	estate := coldEstate()
	priv := "10.0.0.5"
	addrs := []netip.Addr{addr(priv)}
	scope := ColdScope{Addresses: map[netip.Addr]bool{addr(priv): true}}

	internet := []Vantage{{ID: 1, Name: "net", Class: string(custody.ClassInternet)}}
	if jobs := BuildColdJobs(1, estate, addrs, internet, scope); len(jobs) != 0 {
		t.Errorf("a private address must not be probed from an internet-class vantage, got %d jobs", len(jobs))
	}
	internal := []Vantage{{ID: 2, Name: "lan", Class: string(custody.ClassInternal)}}
	if jobs := BuildColdJobs(1, estate, addrs, internal, scope); len(jobs) != 1 {
		t.Errorf("a private address must be probed from an internal-class vantage, got %d jobs", len(jobs))
	}
}

// A third-party address never enters a cold job even if it were (wrongly) opted
// in: Custody refuses it, so the opt-in gate can never smuggle it past.
func TestBuildColdJobsNeverProbesThirdParty(t *testing.T) {
	estate := coldEstate()
	third := "8.8.4.4" // covered by no Seed
	addrs := []netip.Addr{addr("93.184.216.10"), addr(third)}
	vantages := []Vantage{{ID: 1, Name: "net", Class: string(custody.ClassInternet)}}
	scope := ColdScope{
		Addresses: map[netip.Addr]bool{addr("93.184.216.10"): true, addr(third): true},
	}
	jobs := BuildColdJobs(1, estate, addrs, vantages, scope)
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	for _, a := range jobs[0].Addresses {
		if a == third {
			t.Fatalf("third-party address %s appeared in a cold job — Custody gate did not hold", third)
		}
	}
}

// The cold Scan's scope is the full 1–65535 TCP port range, and it dispatches
// the connect-outcome leaf — never a new leaf.
func TestBuildColdJobsFullPortRangeAndConnectOutcome(t *testing.T) {
	estate := coldEstate()
	a := "93.184.216.10"
	addrs := []netip.Addr{addr(a)}
	vantages := []Vantage{{ID: 1, Name: "net", Class: string(custody.ClassInternet)}}
	scope := ColdScope{Addresses: map[netip.Addr]bool{addr(a): true}}

	jobs := BuildColdJobs(1, estate, addrs, vantages, scope)
	if len(jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.Kind != connectoutcome.Kind {
		t.Errorf("cold job dispatches %q, want the reused %q leaf", j.Kind, connectoutcome.Kind)
	}
	if len(j.TCPPorts) != 65535 {
		t.Fatalf("cold covers %d TCP ports, want the full 65535", len(j.TCPPorts))
	}
	if j.TCPPorts[0] != 1 || j.TCPPorts[len(j.TCPPorts)-1] != 65535 {
		t.Errorf("cold port range = [%d..%d], want [1..65535]", j.TCPPorts[0], j.TCPPorts[len(j.TCPPorts)-1])
	}

	// The wire scope is a connect-outcome scope carrying the full range and the
	// shipped safety profile — recorded by content, not defaulted by the leaf.
	spec, err := j.JobSpec("job-1")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != connectoutcome.Kind {
		t.Errorf("spec kind = %q, want %q", spec.Kind, connectoutcome.Kind)
	}
	var sc connectoutcome.Scope
	if err := json.Unmarshal(spec.Scope, &sc); err != nil {
		t.Fatal(err)
	}
	if len(sc.TCPPorts) != 65535 {
		t.Errorf("wire scope carries %d TCP ports, want 65535", len(sc.TCPPorts))
	}
	if sc.Profile.PerHostConnPerSec != 50 {
		t.Errorf("safety profile not carried by content: %+v", sc.Profile)
	}
}

// No addresses and no vantages are both legible empty states, never an error.
func TestBuildColdJobsEmptyInputsAreLegible(t *testing.T) {
	estate := coldEstate()
	scope := ColdScope{Addresses: map[netip.Addr]bool{addr("93.184.216.10"): true}}
	v := []Vantage{{ID: 1, Name: "net", Class: string(custody.ClassInternet)}}
	if jobs := BuildColdJobs(1, estate, nil, v, scope); jobs != nil {
		t.Errorf("no addresses should yield no jobs, got %d", len(jobs))
	}
	if jobs := BuildColdJobs(1, estate, []netip.Addr{addr("93.184.216.10")}, nil, scope); jobs != nil {
		t.Errorf("no vantages should yield no jobs, got %d", len(jobs))
	}
}

// The recorded scope states the full range by content, so a pair never probed
// can never read as an absence we measured; a dead-lettered cold Batch records
// an empty scope instead.
func TestColdScopeRecordAndDeadLetter(t *testing.T) {
	estate := coldEstate()
	a := "93.184.216.10"
	scope := ColdScope{Addresses: map[netip.Addr]bool{addr(a): true}}
	j := BuildColdJobs(1, estate, []netip.Addr{addr(a)}, []Vantage{{ID: 1, Name: "net", Class: string(custody.ClassInternet)}}, scope)[0]

	raw, err := j.AttemptedScope()
	if err != nil {
		t.Fatal(err)
	}
	var rec coldScopeRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatal(err)
	}
	if rec.PortRangeLow != 1 || rec.PortRangeHigh != 65535 {
		t.Errorf("recorded scope range = [%d..%d], want [1..65535]", rec.PortRangeLow, rec.PortRangeHigh)
	}
	if len(rec.Addresses) != 1 || rec.Addresses[0] != a {
		t.Errorf("recorded addresses = %v, want [%s]", rec.Addresses, a)
	}

	empty, err := EmptyColdScope("net")
	if err != nil {
		t.Fatal(err)
	}
	var er coldScopeRecord
	if err := json.Unmarshal(empty, &er); err != nil {
		t.Fatal(err)
	}
	if len(er.Addresses) != 0 {
		t.Errorf("dead-lettered cold scope must be empty, got %v", er.Addresses)
	}
}

func TestColdTCPPortsIsFullRange(t *testing.T) {
	ports := coldTCPPorts()
	if len(ports) != 65535 {
		t.Fatalf("full range has %d ports, want 65535", len(ports))
	}
	if ports[0] != 1 || ports[65534] != 65535 {
		t.Errorf("range bounds = [%d..%d], want [1..65535]", ports[0], ports[65534])
	}
}
