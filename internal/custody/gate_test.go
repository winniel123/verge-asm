package custody

import (
	"net/netip"
	"testing"
)

type probeAttempt struct {
	addr    netip.Addr
	vantage VantageClass
	port    int
	tier    string
	rate    string
}

type prober struct {
	estate   Estate
	attempts []probeAttempt
}

func (p *prober) connect(a probeAttempt) { p.attempts = append(p.attempts, a) }

func (p *prober) run(addrs []netip.Addr, vantages []VantageClass) {
	tiers := map[string][]int{
		"sensitive": {22, 23, 3389, 5432, 6379, 9200, 11211, 27017},
		"top":       {21, 25, 53, 110, 143, 445, 993, 995},
		"web":       {80, 443, 8080, 8443},
	}
	rates := []string{"default", "gentle", "aggressive"}
	for _, a := range addrs {
		for _, vc := range vantages {
			for tier, ports := range tiers {
				for _, port := range ports {
					for _, rate := range rates {
						if !p.estate.MayProbe(a, vc) {
							continue
						}
						p.connect(probeAttempt{a, vc, port, tier, rate})
					}
				}
			}
		}
	}
}

func (p *prober) attemptsFor(a netip.Addr) int {
	n := 0
	for _, at := range p.attempts {
		if at.addr == a.Unmap() {
			n++
		}
	}
	return n
}

var everyClass = []VantageClass{ClassInternet, ClassInternal, ClassUnverified}

func TestThirdPartyRefusedOnEveryPortTierRate(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.0/24")}}
	operatorAddr := addr("52.1.2.10")
	thirdPartyAddr := addr("13.32.1.1")

	p := &prober{estate: e}
	p.run([]netip.Addr{operatorAddr, thirdPartyAddr}, everyClass)

	if p.attemptsFor(thirdPartyAddr) != 0 {
		t.Fatalf("third-party address drew %d probe attempts, want 0", p.attemptsFor(thirdPartyAddr))
	}
	if p.attemptsFor(operatorAddr) == 0 {
		t.Fatal("operator address drew 0 attempts — the prober is not probing, so the zero above is vacuous")
	}
}

func TestNonGloballyReachableDenotationGate(t *testing.T) {
	scoped := &prober{estate: Estate{AddressScopes: []netip.Prefix{cidr("10.0.0.0/24")}}}
	priv := addr("10.0.0.5")
	scoped.run([]netip.Addr{priv}, everyClass)

	for _, at := range scoped.attempts {
		if at.vantage == ClassInternet {
			t.Fatalf("private address probed from internet-class vantage: %+v", at)
		}
	}
	if scoped.attemptsFor(priv) == 0 {
		t.Fatal("private address under an address scope drew 0 attempts from non-internet vantages, want >0")
	}

	extended := &prober{estate: Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: priv}},
	}}
	extended.run([]netip.Addr{priv}, everyClass)
	if extended.attemptsFor(priv) != 0 {
		t.Fatalf("private address under a custody extension alone drew %d attempts, want 0", extended.attemptsFor(priv))
	}
}

func TestCloudMetadataAndLoopbackNeverProbed(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "localhost.example.com", Address: addr("127.0.0.1")},
			{Owner: "meta.example.com", Address: addr("169.254.169.254")},
		},
	}
	p := &prober{estate: e}
	// These measure the prober itself, or pull cloud metadata into http-identity (ADR-0079).
	p.run([]netip.Addr{addr("127.0.0.1"), addr("169.254.169.254")}, everyClass)
	if len(p.attempts) != 0 {
		t.Fatalf("loopback / metadata addresses drew %d attempts, want 0", len(p.attempts))
	}
}

func TestExtensionCoveredGloballyReachableProbedEverywhere(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: addr("52.1.2.3")}},
	}
	for _, vc := range everyClass {
		if !e.MayProbe(addr("52.1.2.3"), vc) {
			t.Errorf("MayProbe(52.1.2.3, %s) = false, want true (globally-reachable operator address)", vc)
		}
	}
}

func TestQueryIsNotAConnect(t *testing.T) {
	e := Estate{}
	if e.MayProbe(addr("52.1.2.3"), ClassInternet) {
		t.Error("an install holding custody of nothing must not probe any address")
	}
	if got := e.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive on a custody-of-nothing install = %q, want third-party", got)
	}
}
