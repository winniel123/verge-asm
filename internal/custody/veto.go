package custody

import "net/netip"

// A measurement may narrow a Derived reach; it never overrules a Declared act (ADR-0129, #956).

type EdgeFanout struct {
	Enabled bool // false is the pre-measurement reach, the loud direction ADR-0129 §2 accepts

	BatchCompleted bool

	// Per limb: a store-wide floor would hold every extension candidate silently (#1018).

	ExtensionErrored bool

	// A missing key is measurement pending, a currency state and never a count band (ADR-0129, #954).

	Shared map[netip.Addr]bool // keys are Unmap'ed, matching every comparison in this package

	// A wholesale walk of a bound Shared reads short, so the census refuses a partial record (#1036).

	Partial bool
}

// No specificity test ranks the limbs: an invented threshold in the safety path is refused (#27).

func (f EdgeFanout) admits(addr netip.Addr) bool {
	// Called from coveredByExtension alone, so the declared-limb keys reach no gate (#988).
	if !f.inForce() {
		return true
	}
	shared, measured := f.Shared[addr]
	// Hold-then-open: an unmeasured edge is held, so a CDN estate shows no first-tick churn.
	if !measured {
		return false
	}
	return !shared
}

func (f EdgeFanout) inForce() bool {
	// One predicate, so the gate and the census cannot read the absence case differently (ADR-0129).
	return f.Enabled && !f.ExtensionErrored
}

func (f EdgeFanout) overExtension(candidates []netip.Addr) EdgeFanout {
	// Cleared on entry: one record read from the store may resolve over two estates (ADR-0188 §2).
	f.ExtensionErrored = false
	// A new candidate set reads as errored: nothing here tells a lag from a failure (ADR-0129 §2).
	if !f.Enabled || !f.BatchCompleted || len(candidates) == 0 {
		return f
	}
	// Every negative outcome records a row, so an errored limb means the Scan measured nothing at all.
	for _, addr := range candidates {
		if _, measured := f.Shared[addr]; measured {
			return f
		}
	}
	// An errored limb reaches: a silent withhold is the direction ADR-0129 §2 refuses.
	f.ExtensionErrored = true
	return f
}
