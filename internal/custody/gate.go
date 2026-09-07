package custody

import "net/netip"

type VantageClass string

const (
	ClassInternet   VantageClass = "internet"
	ClassInternal   VantageClass = "internal"
	ClassUnverified VantageClass = "unverified"
)

func (c VantageClass) IsInternet() bool { return c == ClassInternet }

// No port-, tier- or rate-shaped argument: none of those opens the gate partially (ADR-0019).

func (e Estate) MayProbe(addr netip.Addr, vc VantageClass) bool {
	_, ok := e.ProbeRealm(addr, vc)
	return ok
}

// One read serves the dispatch gate and the socket guard, so the two cannot drift (ADR-0225 §1).

func (e Estate) ProbeRealm(addr netip.Addr, vc VantageClass) (netip.Prefix, bool) {
	addr = addr.Unmap()

	if e.Derive(addr) != Operator {
		return netip.Prefix{}, false
	}

	// A non-globally-reachable address denotes a different machine in every realm (ADR-0079).
	if IsNonGloballyReachable(addr) {
		// Redundant with Derive by design: ADR-0079's realm claim sits where the gate reads it.
		scope, covered := e.coveringAddressScope(addr)
		if !covered {
			return netip.Prefix{}, false
		}
		// Barring internal would delete private-space probing wherever no prober runs (ADR-0079).
		if vc.IsInternet() {
			return netip.Prefix{}, false
		}
		return scope, true
	}

	// A globally reachable address needs no realm, so the guard stays unconditional over it.
	return netip.Prefix{}, true
}

// A Realm holds declared address scopes alone: never a predicate, never a verdict (ADR-0225 §2).

type Realm struct {
	scopes []netip.Prefix
}

// Only a scope ProbeRealm returned enters, so no refused address can be added (ADR-0225 §1).

func (r Realm) With(scope netip.Prefix) Realm {
	if !scope.IsValid() {
		return r
	}
	// Contains reads the prefix bits alone, so masking here changes no membership (#1610).
	scope = scope.Masked()
	for _, p := range r.scopes {
		if p == scope {
			return r
		}
	}
	out := make([]netip.Prefix, len(r.scopes), len(r.scopes)+1)
	copy(out, r.scopes)
	return Realm{scopes: append(out, scope)}
}

func (r Realm) Contains(addr netip.Addr) bool {
	addr = addr.Unmap()
	for _, p := range r.scopes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (r Realm) CIDRs() []string {
	if len(r.scopes) == 0 {
		return nil
	}
	out := make([]string, 0, len(r.scopes))
	for _, p := range r.scopes {
		out = append(out, p.String())
	}
	return out
}

// A malformed entry drops rather than widening: the residue is a refused dial (ADR-0225 §3).

func ParseRealm(cidrs []string) Realm {
	var r Realm
	for _, s := range cidrs {
		p, err := netip.ParsePrefix(s)
		if err != nil {
			continue
		}
		r = r.With(p)
	}
	return r
}
