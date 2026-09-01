package queue

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

func mustAddr(s string) netip.Addr { return netip.MustParseAddr(s) }

// A declared address scope enumerates into the candidate set even when nothing
// resolves into it (#779, ADR-0047): a dark /30 alone yields four targets.
func TestCandidateAddrsEnumeratesDarkScope(t *testing.T) {
	scopes := []netip.Prefix{netip.MustParsePrefix("93.184.216.0/30")}
	got := slices.Collect(candidateAddrs(nil, scopes, nil))
	if len(got) != 4 {
		t.Fatalf("a /30 with no resolutions must yield 4 targets, got %d: %v", len(got), got)
	}
	want := map[string]bool{"93.184.216.0": true, "93.184.216.1": true, "93.184.216.2": true, "93.184.216.3": true}
	for _, a := range got {
		if !want[a.String()] {
			t.Errorf("unexpected enumerated address %s", a)
		}
	}
}

// The union is deduplicated: an address that is both resolved and inside a scope
// appears once, so it is never probed twice.
func TestCandidateAddrsDedupsResolvedAndEnumerated(t *testing.T) {
	resolved := []netip.Addr{mustAddr("93.184.216.1"), mustAddr("8.8.8.8")}
	scopes := []netip.Prefix{netip.MustParsePrefix("93.184.216.0/30")}
	got := slices.Collect(candidateAddrs(resolved, scopes, nil))
	// 8.8.8.8 (resolved, outside the scope) + the four /30 addresses = 5; the
	// resolved 93.184.216.1 that is also inside the scope is not double-counted.
	if len(got) != 5 {
		t.Fatalf("union must dedup resolved∩scope, want 5 targets, got %d: %v", len(got), got)
	}
	seen := map[netip.Addr]int{}
	for _, a := range got {
		seen[a]++
	}
	for a, n := range seen {
		if n != 1 {
			t.Errorf("address %s appears %d times, want 1", a, n)
		}
	}
	// Resolved addresses come first, in order.
	if got[0].String() != "93.184.216.1" || got[1].String() != "8.8.8.8" {
		t.Errorf("resolved addresses must lead the candidate set, got %v", got[:2])
	}
}

// With no scopes declared the candidate set is exactly the resolved addresses —
// the pre-#779 behaviour is preserved when nothing enumerates.
func TestCandidateAddrsNoScopesIsResolvedOnly(t *testing.T) {
	resolved := []netip.Addr{mustAddr("93.184.216.10"), mustAddr("203.0.113.7")}
	got := slices.Collect(candidateAddrs(resolved, nil, nil))
	if len(got) != 2 || got[0].String() != "93.184.216.10" || got[1].String() != "203.0.113.7" {
		t.Fatalf("with no scopes the candidate set is the resolved addresses, got %v", got)
	}
}

// A declared `address` exclusion takes its addresses out of the HOT tier's walk
// (ADR-0133 §3). The hot fan-out enumerates the Estate's own scopes, so the
// predicate and the scopes come from one value here, exactly as fanOutHot builds
// them.
func TestCandidateAddrsSkipsAnExcludedAddressOnTheHotTier(t *testing.T) {
	scopes := []netip.Prefix{netip.MustParsePrefix("93.184.216.0/30")}
	estate := custody.Estate{AddressScopes: scopes}.
		WithAddressExclusions([]netip.Prefix{netip.MustParsePrefix("93.184.216.2/31")})

	got := slices.Collect(candidateAddrs(nil, scopes, estate.AddressExcluded))
	want := []string{"93.184.216.0", "93.184.216.1"}
	if len(got) != len(want) {
		t.Fatalf("an excluded /31 inside a declared /30 must leave 2 targets, got %d: %v", len(got), got)
	}
	for i, a := range got {
		if a.String() != want[i] {
			t.Errorf("target %d is %s, want %s: the excluded range is still walked", i, a, want[i])
		}
	}
}

// The COLD tier needs the same skip, and it needs it through its OWN scope source.
// fanOutCold enumerates coldScope's opted-in prefixes rather than the Estate's
// scopes, so a change that reached only the hot path would leave the cold sweep
// walking the excluded range (ADR-0133 §3). The two prefix sets are deliberately
// different here, which is the drift the row guards.
func TestCandidateAddrsSkipsAnExcludedAddressOnTheColdTier(t *testing.T) {
	estate := custody.Estate{AddressScopes: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}}.
		WithAddressExclusions([]netip.Prefix{netip.MustParsePrefix("198.51.100.2/31")})
	optedIn := []netip.Prefix{netip.MustParsePrefix("198.51.100.0/30")}

	got := slices.Collect(candidateAddrs(nil, optedIn, estate.AddressExcluded))
	want := []string{"198.51.100.0", "198.51.100.1"}
	if len(got) != len(want) {
		t.Fatalf("the cold sweep must skip the excluded /31 of its opted-in /30, want 2 targets, got %d: %v", len(got), got)
	}
	for i, a := range got {
		if a.String() != want[i] {
			t.Errorf("cold target %d is %s, want %s: the cold tier walks the excluded range", i, a, want[i])
		}
	}
}

// The exclusion cuts the `Seed` limb ALONE, so it never removes a RESOLVED address
// from the walk (ADR-0133 §1). A custody extension may reach that address, in which
// case it is still an operator address and is still probed; the gate decides, and
// this walk must not decide it first.
func TestCandidateAddrsKeepsAnExcludedResolvedAddress(t *testing.T) {
	resolved := []netip.Addr{mustAddr("198.51.100.5")}
	estate := custody.Estate{}.
		WithAddressExclusions([]netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")})

	got := slices.Collect(candidateAddrs(resolved, nil, estate.AddressExcluded))
	if len(got) != 1 || got[0].String() != "198.51.100.5" {
		t.Fatalf("an excluded RESOLVED address must stay in the walk, got %v: the exclusion removed more than the declaration added", got)
	}
}

// Multiple declared scopes each enumerate, and their addresses union without
// duplication where the scopes are disjoint.
func TestCandidateAddrsUnionsMultipleScopes(t *testing.T) {
	scopes := []netip.Prefix{
		netip.MustParsePrefix("93.184.216.0/31"),
		netip.MustParsePrefix("198.51.100.0/31"),
	}
	got := slices.Collect(candidateAddrs(nil, scopes, nil))
	if len(got) != 4 {
		t.Fatalf("two /31 scopes enumerate 4 addresses, got %d: %v", len(got), got)
	}
}
