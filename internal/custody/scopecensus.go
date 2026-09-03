package custody

import "net/netip"

// A measurement contradicting a declaration, held and not shown, fails silently (ADR-0013, #989).

type AddressScopeCensusEntry struct { // display only: no gate may read it (ADR-0129 §5)
	Scope       netip.Prefix
	SharedEdges int // a count of the operator's addresses, never the threshold (ADR-0013's nag test)
}

func (e Estate) AddressScopeCensus() []AddressScopeCensusEntry {
	// A bound read leaves this wholesale walk short with nothing to say so (#1036).
	if !e.edgeFanout.Enabled || e.edgeFanout.Partial {
		// The errored floor decides one limb's reach, so the test reads Enabled, never inForce (#1018).
		return nil
	}
	scopes := make([]netip.Prefix, 0, len(e.AddressScopes))
	counts := make(map[netip.Prefix]int, len(e.AddressScopes))
	for _, p := range e.AddressScopes {
		if _, dup := counts[p]; dup {
			continue
		}
		counts[p] = 0
		scopes = append(scopes, p)
	}
	// Walking the measurement and not the scope: a declared scope can be an IPv4 /8 (ADR-0127).
	for addr, shared := range e.edgeFanout.Shared {
		if !shared {
			continue
		}
		if e.AddressExcluded(addr) {
			continue // the remedy the row names, taken — filtered on READ, pruned never
		}
		// Both overlapping scopes count it: the remedy is declared per scope (ADR-0129 §5).
		for _, p := range scopes {
			if p.Contains(addr) {
				counts[p]++
			}
		}
	}
	var out []AddressScopeCensusEntry
	for _, p := range scopes {
		// A pending row on every declared address on day one would be noise, not a census (#956).
		if counts[p] == 0 {
			continue
		}
		out = append(out, AddressScopeCensusEntry{Scope: p, SharedEdges: counts[p]})
	}
	return out
}
