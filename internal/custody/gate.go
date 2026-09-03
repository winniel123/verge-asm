package custody

import "net/netip"

// VantageClass is which side of the operator's boundary a Vantage sits on
// (CONTEXT.md `Vantage class`). The gate reads only whether a class is
// `internet`; the denotation precondition bars a non-globally-reachable address
// from every `internet`-class Vantage and admits it from any class that is not
// `internet` (ADR-0079's *not internet*, deliberately not *internal*: `internal`
// is provable only where an external prober observed the instance, so barring on
// it would delete internal probing of private space on every install without a
// prober).
type VantageClass string

const (
	ClassInternet   VantageClass = "internet"
	ClassInternal   VantageClass = "internal"
	ClassUnverified VantageClass = "unverified"
)

func (c VantageClass) IsInternet() bool { return c == ClassInternet }

// MayProbe is the total probing gate: whether an active probe may be attempted
// against addr from a Vantage of class vc. It is the single question a prober
// asks before any connect, on any port, of any tier, at any rate — there is no
// port-, protocol- or rate-shaped argument, because none of those opens it
// partially (ADR-0019). A false answer is a total block over the address.
//
// Two tests run in series and they ask different things (ADR-0079): denotation
// asks *of what machine?* and authority asks *may we?*.
//
//   - Authority. Custody must be `operator`. A `third-party` address is refused
//     outright. This is the whole of the gate for a globally-reachable address.
//   - Denotation. A non-globally-reachable address denotes one machine per
//     realm, so it is connected to only where a declared ADDRESS SCOPE covers it
//     (the operator's own realm claim) and only from a Vantage that is not
//     `internet`-class. A custody extension alone never opens this — it declares
//     no realm.
//
// A query (resolution / dns-record) is not a connect and is not gated here; this
// gate is over an active probe against an Address.
func (e Estate) MayProbe(addr netip.Addr, vc VantageClass) bool {
	addr = addr.Unmap()

	if e.Derive(addr) != Operator {
		return false
	}

	// Denotation precondition — only reached by an operator address, and only
	// binding when the address is non-globally-reachable.
	if IsNonGloballyReachable(addr) {
		// An operator non-globally-reachable address is operator only via an
		// address scope (an extension never covers one), so this check is a
		// stated belt-and-braces restatement of ADR-0079's *a declared address
		// scope covers it*.
		if !e.coveredByAddressScope(addr) {
			return false
		}
		if vc.IsInternet() {
			return false
		}
	}

	return true
}
