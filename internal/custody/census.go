package custody

import "net/netip"

// There is no reached state — a reached edge is an ordinary covered address (ADR-0129 #944).

type ExtensionState string

const (
	ExtensionDeclined ExtensionState = "declined"
	ExtensionPending  ExtensionState = "pending"
)

type ExtensionCensusEntry struct {
	Name    string
	Address netip.Addr
	State   ExtensionState
	Scope   netip.Prefix
}

func (e Estate) ExtensionCensus() []ExtensionCensusEntry {
	// Out of force the extension reaches, so a pending row names a hold that is not happening (#1018).
	if !e.edgeFanout.inForce() {
		return nil
	}
	type pair struct {
		name string
		addr netip.Addr
	}
	var out []ExtensionCensusEntry
	seen := make(map[pair]struct{}, len(e.Resolutions))
	for _, r := range e.Resolutions {
		addr := r.Address.Unmap()
		if IsNonGloballyReachable(addr) || !e.withinExtendedZone(r.Owner) {
			continue
		}
		shared, measured := e.edgeFanout.Shared[addr]
		if measured && !shared {
			continue
		}
		key := pair{name: r.Owner, addr: addr}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		state := ExtensionPending
		if measured {
			state = ExtensionDeclined
		}
		// A dual-limb row states both limbs; a bare declined would read as a contradiction (#956).
		scope, _ := e.coveringAddressScope(addr)
		out = append(out, ExtensionCensusEntry{
			Name: r.Owner, Address: addr, State: state, Scope: scope,
		})
	}
	return out
}
