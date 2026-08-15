// Package estate composes the membership vector — the derivation that decides
// which Names and cited Addresses are in the estate. Membership reads the
// `resolution` facet, which **two** leaves decide jointly: `resolution-walk`
// decides `NameError │ NoData │ Lame │ Resolved`, and `wildcard-discrimination`
// decides `Shadowed`. A derivation composes every leaf that decides the value it
// reads (ADR-0086), so membership is never decided by one leaf alone where both
// apply — a bump of either leaf moves the value membership reads.
//
// `Shadowed` cites no `Address`, so the verdict decides whether an answer's
// address set enters the estate: a `Shadowed` Name is not withdrawn (it stays,
// admitted by its Citation rather than by this answer), but every `Address` held
// only by that citation leaves the estate. This package extends the one
// membership path additively so a `Shadowed` Name is suppressed; it does not fork
// a second one.
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

// FinalResolution is the one recorded `resolution` value for a (Name, Vantage) —
// the value resolution-walk and wildcard-discrimination decide jointly. It is
// what Compose produces and what Membership reads.
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
// value per (Name, Vantage) that Membership reads.
type Observation struct {
	Name       string
	Vantage    string
	Resolution FinalResolution
}

// Estate is the membership census: the Names present, and the Addresses a current
// resolution cites. Both are the output of composing every leaf that decides the
// value read.
type Estate struct {
	Names     []string
	Addresses []string
}

// Membership computes the estate from the latest composed resolution per (Name,
// Vantage) and the Seed-covered Names (Declared, carrying no vector, so always
// present). A presence read is existential within the witnesses (ADR-0080): a
// Name is present where **some** vantage's current resolution does not withdraw
// it, and it withdraws only where **every** available vantage reads `NameError`.
// A `Shadowed` answer never withdraws a Name and never cites an Address, which is
// how a repointed wildcard's fictional names stay out of the estate while the
// real one beneath it holds by its Citation.
func Membership(latest []Observation, seedCovered []string) Estate {
	// Per Name: present unless every vantage withdrew it (NameError). Track
	// whether any vantage was seen at all, so a Name with no observation is
	// decided by Seed coverage alone.
	sawVantage := map[string]bool{}
	withdrawnEverywhere := map[string]bool{}
	citedAddrs := map[string]struct{}{}

	for _, o := range latest {
		name := o.Name
		if !sawVantage[name] {
			sawVantage[name] = true
			withdrawnEverywhere[name] = true
		}
		if o.Resolution.Outcome != OutcomeNameError {
			withdrawnEverywhere[name] = false
		}
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
	for name := range sawVantage {
		if !withdrawnEverywhere[name] {
			present[name] = struct{}{}
		}
	}
	// A Seed-covered Name is Declared and in the estate regardless of resolution.
	for _, name := range seedCovered {
		present[name] = struct{}{}
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
