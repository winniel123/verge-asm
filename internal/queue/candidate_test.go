package queue

import (
	"net/netip"
	"slices"
	"testing"

	"github.com/winniel123/verge-asm/internal/custody"
)

func mustAddr(s string) netip.Addr { return netip.MustParseAddr(s) }

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

func TestCandidateAddrsDedupsResolvedAndEnumerated(t *testing.T) {
	resolved := []netip.Addr{mustAddr("93.184.216.1"), mustAddr("8.8.8.8")}
	scopes := []netip.Prefix{netip.MustParsePrefix("93.184.216.0/30")}
	got := slices.Collect(candidateAddrs(resolved, scopes, nil))
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
	if got[0].String() != "93.184.216.1" || got[1].String() != "8.8.8.8" {
		t.Errorf("resolved addresses must lead the candidate set, got %v", got[:2])
	}
}

func TestCandidateAddrsNoScopesIsResolvedOnly(t *testing.T) {
	resolved := []netip.Addr{mustAddr("93.184.216.10"), mustAddr("203.0.113.7")}
	got := slices.Collect(candidateAddrs(resolved, nil, nil))
	if len(got) != 2 || got[0].String() != "93.184.216.10" || got[1].String() != "203.0.113.7" {
		t.Fatalf("with no scopes the candidate set is the resolved addresses, got %v", got)
	}
}

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

func TestCandidateAddrsKeepsAnExcludedResolvedAddress(t *testing.T) {
	resolved := []netip.Addr{mustAddr("198.51.100.5")}
	estate := custody.Estate{}.
		WithAddressExclusions([]netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")})

	got := slices.Collect(candidateAddrs(resolved, nil, estate.AddressExcluded))
	if len(got) != 1 || got[0].String() != "198.51.100.5" {
		t.Fatalf("an excluded RESOLVED address must stay in the walk, got %v: the exclusion removed more than the declaration added", got)
	}
}

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
