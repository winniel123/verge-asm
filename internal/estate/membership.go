// Package estate composes the membership vector — the derivation that decides
// which Names and cited Addresses are in the estate. Membership reads the
// `resolution` facet, which **two** leaves decide jointly: `resolution-walk`
// decides `NameError │ NoData │ Lame │ Resolved`, and `wildcard-discrimination`
// decides `Shadowed`. A derivation composes every leaf that decides the value it
// reads (ADR-0086), so membership is never decided by one leaf alone where both
// apply — a bump of either leaf moves the value membership reads.
//
// `Shadowed` cites no `Address` and suppresses the affected `Name` "as
// affirmatively as" `resolution-walk`'s own `NameError` (#192 AC): a Name read
// `Shadowed` at every available vantage leaves the estate, and every `Address`
// held only by that answer leaves with it. Suppression is existential like every
// presence read — a Name still admitted by some vantage (Resolved / NoData /
// Lame), by a Citation, or by a Seed stays. This package extends the one
// membership path additively; it does not fork a second one.
package estate

import (
	"sort"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// The composed resolution outcomes membership reads. The first four are
// resolution-walk's; Shadowed is wildcard-discrimination's; Gap is either leaf's
// "we could not say" and never a withdrawal.
const (
	OutcomeResolved  = "Resolved"
	OutcomeNoData    = "NoData"
	OutcomeNameError = "NameError"
	OutcomeLame      = "Lame"
	OutcomeShadowed  = "Shadowed"
	OutcomeGap       = "Gap"
)

type FinalResolution struct {
	Outcome   string
	Addresses []string
}

// Compose folds resolution-walk's outcome and wildcard-discrimination's verdict
// into the one recorded resolution value, inside one Batch (ADR-0011). This is
// the site where both leaves meet: `Shadowed` overwrites whatever resolution-walk
// measured and cites nothing; a `Gap` from an incomplete control probe overwrites
// it too and cites nothing; otherwise resolution-walk's value — and its address
// set, where it has one — stands.
func Compose(rwOutcome string, rwAddresses []string, verdict wd.Verdict) FinalResolution {
	switch verdict {
	case wd.VerdictShadowed:
		return FinalResolution{Outcome: OutcomeShadowed}
	case wd.VerdictGap:
		return FinalResolution{Outcome: OutcomeGap}
	default:
		return FinalResolution{Outcome: rwOutcome, Addresses: append([]string(nil), rwAddresses...)}
	}
}

// Observation is one Name's composed resolution at one Vantage — the latest such
// value per (Name, Vantage) that Membership reads. Class names the Vantage class
// the vantage sits in; an empty Class is one default class, the modal
// single-class install, so the cross-class withdrawal rule collapses to the
// single-class case with no special path (ADR-0080).
type Observation struct {
	Name       string
	Vantage    string
	Class      string
	Resolution FinalResolution
}

type Estate struct {
	Names     []string
	Addresses []string
}

// Membership computes the estate from the latest composed resolution per (Name,
// Vantage), the Seed-covered Names, and the Seed-covered Addresses (all Declared,
// carrying no vector, so always present). A presence read is existential within
// the witnesses (ADR-0080): a Name is present where **some** vantage's current
// resolution admits it, and it is suppressed only where **every** available
// vantage reads `NameError` or `Shadowed` (WithdrawnCrossClass). A `Shadowed`
// answer cites no Address and does not admit its Name, which is how a repointed
// wildcard's fictional names stay out of the estate while a real one still
// admitted by another vantage, a Citation, or a Seed holds.
//
// Address membership is the disjunction of its two limbs (AddressPresent): an
// Address is in the estate exactly while a current resolution cites it OR a Seed's
// address scope covers it. seedCoveredAddresses carries the second limb — the
// addresses an address-scope Seed enumerates — so a Seed-covered address is
// present before anything resolves to it, and one held only by a superseded
// `Resolved` leaves unless a Seed still covers it.
func Membership(latest []Observation, seedCoveredNames, seedCoveredAddresses []string) Estate {
	// Per Name, gather the composed outcome at each available (class, vantage) and
	// the Addresses a current Resolved cites. Withdrawal is decided by the one
	// cross-class composition (WithdrawnCrossClass), so a Name is present unless
	// that predicate concludes it left — never by a survivor-only reading here.
	sawName := map[string]bool{}
	perName := map[string][]classedOutcome{}
	citedAddrs := map[string]struct{}{}

	for _, o := range latest {
		name := o.Name
		sawName[name] = true
		perName[name] = append(perName[name], classedOutcome{class: o.Class, outcome: o.Resolution.Outcome})
		// Only a Resolved value cites Addresses; Shadowed, NoData, Lame, NameError
		// and Gap cite nothing, so an Address held only by a superseded Resolved
		// leaves the estate.
		if o.Resolution.Outcome == OutcomeResolved {
			for _, a := range o.Resolution.Addresses {
				citedAddrs[a] = struct{}{}
			}
		}
	}

	present := map[string]struct{}{}
	for name := range sawName {
		if !WithdrawnCrossClass(witnessesByClass(perName[name])) {
			present[name] = struct{}{}
		}
	}
	// A Seed-covered Name is Declared and in the estate regardless of resolution.
	for _, name := range seedCoveredNames {
		present[name] = struct{}{}
	}

	// The Address estate is the disjunction of the two limbs: the addresses a
	// current resolution cites, unioned with the addresses a Seed's address scope
	// covers. A Seed-covered address is present even where no resolution cites it.
	for _, a := range seedCoveredAddresses {
		citedAddrs[a] = struct{}{}
	}

	return Estate{Names: sortedKeys(present), Addresses: sortedSet(citedAddrs)}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedSet(m map[string]struct{}) []string { return sortedKeys(m) }
