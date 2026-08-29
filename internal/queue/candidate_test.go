package queue

import (
	"net/netip"
	"testing"
)

func mustAddr(s string) netip.Addr { return netip.MustParseAddr(s) }

// A declared address scope enumerates into the candidate set even when nothing
// resolves into it (#779, ADR-0047): a dark /30 alone yields four targets.
func TestCandidateAddrsEnumeratesDarkScope(t *testing.T) {
	scopes := []netip.Prefix{netip.MustParsePrefix("93.184.216.0/30")}
	got := candidateAddrs(nil, scopes)
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
	got := candidateAddrs(resolved, scopes)
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
	got := candidateAddrs(resolved, nil)
	if len(got) != 2 || got[0].String() != "93.184.216.10" || got[1].String() != "203.0.113.7" {
		t.Fatalf("with no scopes the candidate set is the resolved addresses, got %v", got)
	}
}

// Multiple declared scopes each enumerate, and their addresses union without
// duplication where the scopes are disjoint.
func TestCandidateAddrsUnionsMultipleScopes(t *testing.T) {
	scopes := []netip.Prefix{
		netip.MustParsePrefix("93.184.216.0/31"),
		netip.MustParsePrefix("198.51.100.0/31"),
	}
	got := candidateAddrs(nil, scopes)
	if len(got) != 4 {
		t.Fatalf("two /31 scopes enumerate 4 addresses, got %d: %v", len(got), got)
	}
}
