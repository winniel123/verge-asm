// Package custody derives control of the listener from confirmed Seeds alone — never from
// registry expansion or an unconfirmed proposal (ADR-0013, ADR-0012). The gate it holds is
// total over an address: no port, no tier, no rate, no carve-out (ADR-0019).
package custody

import (
	"net/netip"
	"strings"
)

// No unknown value: Seed coverage is total, with no lookup left to fail (ADR-0013 §2).

type Custody string

const (
	Operator   Custody = "operator"
	ThirdParty Custody = "third-party"
)

type Resolution struct {
	Owner   string
	Address netip.Addr
}

type Estate struct {
	AddressScopes     []netip.Prefix
	ExtendedZones     []string
	Resolutions       []Resolution
	edgeFanout        EdgeFanout
	addressExclusions []netip.Prefix
}

func (e Estate) WithAddressExclusions(prefixes []netip.Prefix) Estate {
	// Containment compares the prefix-length bits alone, so host bits in a prefix need no masking.
	e.addressExclusions = prefixes
	return e
}

// Exported because the cold tier enumerates its own prefixes and never these scopes (ADR-0133 §3).

func (e Estate) AddressExcluded(addr netip.Addr) bool {
	// True is not a probing verdict: an excluded address a custody extension reaches is still probed.
	addr = addr.Unmap()
	for _, p := range e.addressExclusions {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (e Estate) WithEdgeFanout(f EdgeFanout) Estate {
	// The floor is resolved here because only the assembler holds the candidate set (#1018).
	e.edgeFanout = f.overExtension(e.ExtensionCandidates())
	return e
}

func (e Estate) Derive(addr netip.Addr) Custody {
	addr = addr.Unmap()
	if e.coveredByAddressScope(addr) {
		return Operator
	}
	if e.coveredByExtension(addr) {
		return Operator
	}
	return ThirdParty
}

// Declared scopes alone decide a vantage's side: an extension here corrupts the class (CONTEXT.md).

func (e Estate) CoversAddressScope(addr netip.Addr) bool {
	// One matcher, and the ADR-0133 §4 narrowing is intended: never add a second predicate (#711).
	return e.coveredByAddressScope(addr.Unmap())
}

func (e Estate) coveredByAddressScope(addr netip.Addr) bool {
	_, covered := e.coveringAddressScope(addr)
	return covered
}

func (e Estate) coveringAddressScope(addr netip.Addr) (netip.Prefix, bool) {
	// Not in MayProbe: an excluded address an extension reaches still derives operator (ADR-0133 §1).
	if e.AddressExcluded(addr) {
		return netip.Prefix{}, false
	}
	// Overlapping scopes derive the same value, so first match suffices; specificity is never tested.
	for _, p := range e.AddressScopes {
		if p.Contains(addr) {
			return p, true
		}
	}
	return netip.Prefix{}, false
}

func (e Estate) coveredByExtension(addr netip.Addr) bool {
	return e.extensionReaches(addr) && e.edgeFanout.admits(addr)
}

func (e Estate) extensionReaches(addr netip.Addr) bool {
	// An extension declares no realm, so it may not reach a non-globally-reachable address (ADR-0079).
	if IsNonGloballyReachable(addr) {
		return false
	}
	// A CNAME puts the A record on the foreign owner, which is inside no extended zone (ADR-0013 §3).
	for _, r := range e.Resolutions {
		if r.Address.Unmap() != addr {
			continue
		}
		if e.withinExtendedZone(r.Owner) {
			return true
		}
	}
	return false
}

func (e Estate) withinExtendedZone(name string) bool {
	return WithinAnyZone(name, e.ExtendedZones)
}

func WithinAnyZone(name string, zones []string) bool {
	// Every zone question routes here rather than re-deriving a raw HasSuffix on a name (ADR-0055).
	for _, zone := range zones {
		if LabelSuffix(name, zone) {
			return true
		}
	}
	return false
}

func LabelSuffix(candidate, zone string) bool {
	// Label-wise, never string HasSuffix: evilexample.com does not read as inside example.com.
	cl := labels(candidate)
	zl := labels(zone)
	if len(zl) == 0 || len(cl) < len(zl) {
		return false
	}
	off := len(cl) - len(zl)
	for i := range zl {
		if cl[off+i] != zl[i] {
			return false
		}
	}
	return true
}

func labels(name string) []string {
	trimmed := strings.TrimSuffix(name, ".")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(asciiLower(trimmed), ".")
	return parts
}

func asciiLower(s string) string {
	// The protocol folds the 26 ASCII letters alone, so strings.ToLower folds too much (ADR-0055).
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
