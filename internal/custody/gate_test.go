package custody

import (
	"net/netip"
	"testing"
)

// This file proves the gate holds the way the ticket requires: not by observing
// zero *successful* probes, but by driving a prober over fixture data and
// asserting zero *attempts* against a barred address. The prober consults
// MayProbe before every connect, and connect is the only place an attempt is
// recorded — so a barred address that reaches connect zero times was blocked
// before any packet, across the whole port/tier/rate/vantage matrix.

// probeAttempt records one connect the prober was about to make on the wire.
type probeAttempt struct {
	addr    netip.Addr
	vantage VantageClass
	port    int
	tier    string
	rate    string
}

// prober walks a probe plan and gates every connect. connect is the sole egress
// point; recording the attempt there is what makes "zero attempts" a fact about
// the gate rather than about the network.
type prober struct {
	estate   Estate
	attempts []probeAttempt
}

func (p *prober) connect(a probeAttempt) { p.attempts = append(p.attempts, a) }

// run enumerates the whole matrix the gate is total over: every address, every
// vantage class, every port of every tier, at every rate. ADR-0019 says none of
// port, tier or rate opens a closed gate partially — so a barred address must
// draw zero attempts across all of it, and MayProbe takes no such argument.
func (p *prober) run(addrs []netip.Addr, vantages []VantageClass) {
	tiers := map[string][]int{
		// Two of these are the sensitive tiers a stranger's host must never see.
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

// TestThirdPartyRefusedOnEveryPortTierRate is AC-2: a third-party address is
// connected to on no port, by no tier, at any rate — proven by zero attempts.
// The estate also holds an operator address, so the non-zero count there proves
// the prober actually probes and the zero is discrimination, not a dead harness.
func TestThirdPartyRefusedOnEveryPortTierRate(t *testing.T) {
	e := Estate{AddressScopes: []netip.Prefix{cidr("52.1.2.0/24")}}
	operatorAddr := addr("52.1.2.10")   // inside the scope
	thirdPartyAddr := addr("13.32.1.1") // a CDN edge — covered by nothing

	p := &prober{estate: e}
	p.run([]netip.Addr{operatorAddr, thirdPartyAddr}, everyClass)

	if p.attemptsFor(thirdPartyAddr) != 0 {
		t.Fatalf("third-party address drew %d probe attempts, want 0", p.attemptsFor(thirdPartyAddr))
	}
	if p.attemptsFor(operatorAddr) == 0 {
		t.Fatal("operator address drew 0 attempts — the prober is not probing, so the zero above is vacuous")
	}
}

// TestNonGloballyReachableDenotationGate is AC-3. A non-globally-reachable
// address is connected to only where a declared ADDRESS SCOPE covers it and only
// from a non-internet-class Vantage; a custody extension alone does not open it.
func TestNonGloballyReachableDenotationGate(t *testing.T) {
	// Route that opens it: a declared address scope over private space.
	scoped := &prober{estate: Estate{AddressScopes: []netip.Prefix{cidr("10.0.0.0/24")}}}
	priv := addr("10.0.0.5")
	scoped.run([]netip.Addr{priv}, everyClass)

	// From the internet class: zero, always — no realm claim puts an internet
	// vantage in the same realm as a private address.
	for _, at := range scoped.attempts {
		if at.vantage == ClassInternet {
			t.Fatalf("private address probed from internet-class vantage: %+v", at)
		}
	}
	// From internal and unverified: attempted, because the address scope is the
	// operator's realm claim and the class is not internet.
	if scoped.attemptsFor(priv) == 0 {
		t.Fatal("private address under an address scope drew 0 attempts from non-internet vantages, want >0")
	}

	// Route that does NOT open it: a custody extension over the same private
	// address. It is third-party (extension never covers an NGR hop) AND the
	// denotation precondition finds no address scope — zero from every vantage.
	extended := &prober{estate: Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []Resolution{{Owner: "api.example.com", Address: priv}},
	}}
	extended.run([]netip.Addr{priv}, everyClass)
	if extended.attemptsFor(priv) != 0 {
		t.Fatalf("private address under a custody extension alone drew %d attempts, want 0", extended.attemptsFor(priv))
	}
}

// TestCloudMetadataAndLoopbackNeverProbed pins the worked cases ADR-0079 §"Worked"
// calls out: a name resolving to 127.0.0.1 or 169.254.169.254 under an extension
// is never connected to, so the prober never measures itself or retrieves cloud
// instance metadata into http-identity.
func TestCloudMetadataAndLoopbackNeverProbed(t *testing.T) {
	e := Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions: []Resolution{
			{Owner: "localhost.example.com", Address: addr("127.0.0.1")},
			{Owner: "meta.example.com", Address: addr("169.254.169.254")},
		},
	}
	p := &prober{estate: e}
	p.run([]netip.Addr{addr("127.0.0.1"), addr("169.254.169.254")}, everyClass)
	if len(p.attempts) != 0 {
		t.Fatalf("loopback / metadata addresses drew %d attempts, want 0", len(p.attempts))
	}
}

// TestExtensionCoveredGloballyReachableProbedEverywhere confirms the extension's
// motivating case is untouched: a globally-reachable AWS address a custody
// extension covers is operator and is probed from every vantage class — the
// denotation precondition binds only non-globally-reachable addresses.
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

// TestQueryIsNotAConnect: the gate is over an active probe against an Address; it
// says nothing about resolution / dns-record, which run at full aperture on every
// Name regardless of custody. This test documents that boundary by asserting the
// gate is the only thing MayProbe governs — a third-party address returns false
// here, while the DNS facets (not gated by this package) are unaffected.
func TestQueryIsNotAConnect(t *testing.T) {
	e := Estate{} // custody of nothing
	if e.MayProbe(addr("52.1.2.3"), ClassInternet) {
		t.Error("an install holding custody of nothing must not probe any address")
	}
	// Custody derivation itself is total and never errors — the DNS-only install
	// still derives a value for every address it resolves.
	if got := e.Derive(addr("52.1.2.3")); got != ThirdParty {
		t.Errorf("Derive on a custody-of-nothing install = %q, want third-party", got)
	}
}
