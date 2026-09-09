package scan

import (
	"iter"
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/vergecore"
)

func seqOf(addrs ...netip.Addr) iter.Seq[netip.Addr] {
	return func(yield func(netip.Addr) bool) {
		for _, a := range addrs {
			if !yield(a) {
				return
			}
		}
	}
}

func TestHotJobCarriesTheDeclaredScopeThatAdmittedIt(t *testing.T) {
	estate := custody.Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}
	// The shipped local vantage presents no address, so the class is unverified (ADR-0079).
	vantages := []Vantage{{ID: 1, Name: "local"}}

	var jobs []HotJob
	for j := range BuildHotJobs(7, estate, seqOf(netip.MustParseAddr("10.0.0.5")), vantages, vergecore.Default()) {
		jobs = append(jobs, j)
	}
	if len(jobs) != 1 {
		t.Fatalf("BuildHotJobs yielded %d jobs, want 1", len(jobs))
	}
	spec, err := jobs[0].JobSpec("batch")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if !slices.Equal(spec.Realm, []string{"10.0.0.0/24"}) {
		t.Fatalf("JobSpec.Realm = %v, want [10.0.0.0/24]", spec.Realm)
	}
	if custody.EgressGuard("connectoutcome", custody.ParseRealm(spec.Realm))("tcp", "10.0.0.5:80", nil) != nil {
		t.Error("the guard refuses the address the job it was dispatched with declares")
	}
}

func TestHotJobOverAGloballyReachableAddressCarriesNoRealm(t *testing.T) {
	estate := custody.Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("93.184.216.0/24")}}
	vantages := []Vantage{{ID: 1, Name: "local"}}

	var jobs []HotJob
	for j := range BuildHotJobs(7, estate, seqOf(netip.MustParseAddr("93.184.216.34")), vantages, vergecore.Default()) {
		jobs = append(jobs, j)
	}
	if len(jobs) != 1 {
		t.Fatalf("BuildHotJobs yielded %d jobs, want 1", len(jobs))
	}
	spec, err := jobs[0].JobSpec("batch")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	// A realm on a public target would exempt private space for no reason (ADR-0225 §2).
	if len(spec.Realm) != 0 {
		t.Errorf("JobSpec.Realm = %v, want empty", spec.Realm)
	}
}

func TestADiscoveredPrivateAddressGetsNoHotJob(t *testing.T) {
	discovered := netip.MustParseAddr("192.168.1.7")
	estate := custody.Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions:   []custody.Resolution{{Owner: "api.example.com", Address: discovered}},
	}
	vantages := []Vantage{{ID: 1, Name: "local"}}

	for range BuildHotJobs(7, estate, seqOf(discovered), vantages, vergecore.Default()) {
		t.Fatal("BuildHotJobs dispatched a job for an address inside no declared scope")
	}
}

func TestReachedServiceJobsCarryTheDeclaredScope(t *testing.T) {
	estate := custody.Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}}
	vantages := []Vantage{{ID: 1, Name: "local"}}
	services := []ReachedService{
		{VantageID: 1, Address: "10.0.0.5", Port: 443},
		{VantageID: 1, Address: "10.0.0.6", Port: 80},
	}

	tlsJobs := BuildTLSAcceptanceJobs(7, estate, services, vantages)
	if len(tlsJobs) != 1 {
		t.Fatalf("BuildTLSAcceptanceJobs yielded %d jobs, want 1", len(tlsJobs))
	}
	tlsSpec, err := tlsJobs[0].JobSpec("batch")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if !slices.Equal(tlsSpec.Realm, []string{"10.0.0.0/24"}) {
		t.Errorf("tls-acceptance JobSpec.Realm = %v, want [10.0.0.0/24]", tlsSpec.Realm)
	}

	httpJobs := BuildHTTPIdentityJobs(7, estate, services, vantages)
	if len(httpJobs) != 1 {
		t.Fatalf("BuildHTTPIdentityJobs yielded %d jobs, want 1", len(httpJobs))
	}
	httpSpec, err := httpJobs[0].JobSpec("batch")
	if err != nil {
		t.Fatalf("JobSpec: %v", err)
	}
	if !slices.Equal(httpSpec.Realm, []string{"10.0.0.0/24"}) {
		t.Errorf("http-identity JobSpec.Realm = %v, want [10.0.0.0/24]", httpSpec.Realm)
	}
}

func TestReachedServiceJobsOverADiscoveredPrivateAddressAreNotDispatched(t *testing.T) {
	discovered := "192.168.1.7"
	estate := custody.Estate{
		AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")},
		ExtendedZones: []string{"example.com"},
		Resolutions:   []custody.Resolution{{Owner: "api.example.com", Address: netip.MustParseAddr(discovered)}},
	}
	vantages := []Vantage{{ID: 1, Name: "local"}}
	services := []ReachedService{{VantageID: 1, Address: discovered, Port: 443}}

	if got := BuildTLSAcceptanceJobs(7, estate, services, vantages); len(got) != 0 {
		t.Errorf("BuildTLSAcceptanceJobs yielded %d jobs for a discovered private address, want 0", len(got))
	}
	if got := BuildHTTPIdentityJobs(7, estate, services, vantages); len(got) != 0 {
		t.Errorf("BuildHTTPIdentityJobs yielded %d jobs for a discovered private address, want 0", len(got))
	}
}
