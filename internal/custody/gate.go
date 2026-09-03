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
	addr = addr.Unmap()

	if e.Derive(addr) != Operator {
		return false
	}

	// A non-globally-reachable address denotes a different machine in every realm (ADR-0079).
	if IsNonGloballyReachable(addr) {
		// Deliberately redundant with Derive: ADR-0079's realm claim is stated where the gate reads it.
		if !e.coveredByAddressScope(addr) {
			return false
		}
		// Barring on internal would delete private-space probing wherever no prober runs (ADR-0079).
		if vc.IsInternet() {
			return false
		}
	}

	return true
}
