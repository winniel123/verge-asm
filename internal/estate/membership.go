// Package estate composes the membership vector: which Names and cited Addresses
// are in the estate. It composes every leaf that decides the `resolution` facet it
// reads, and a presence read is existential (ADR-0086, ADR-0080, #192).
package estate

import (
	"sort"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// The first four outcomes are resolution-walk's, Shadowed is wildcard-discrimination's (ADR-0086).

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

func Compose(rwOutcome string, rwAddresses []string, verdict wd.Verdict) FinalResolution {
	// Both leaves' values must come from one Batch, or the fold compares moments (ADR-0011).
	switch verdict {
	case wd.VerdictShadowed:
		return FinalResolution{Outcome: OutcomeShadowed}
	case wd.VerdictGap:
		return FinalResolution{Outcome: OutcomeGap}
	default:
		return FinalResolution{Outcome: rwOutcome, Addresses: append([]string(nil), rwAddresses...)}
	}
}

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

func Membership(latest []Observation, seedCoveredNames, seedCoveredAddresses []string) Estate {
	// Presence is never decided by a survivor-only reading here; one predicate decides (ADR-0080).
	sawName := map[string]bool{}
	perName := map[string][]classedOutcome{}
	citedAddrs := map[string]struct{}{}

	for _, o := range latest {
		name := o.Name
		sawName[name] = true
		perName[name] = append(perName[name], classedOutcome{class: o.Class, outcome: o.Resolution.Outcome})
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
	for _, name := range seedCoveredNames {
		present[name] = struct{}{}
	}

	// The Seed limb is not redundant: a declared address is a subject before anything resolves.
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
