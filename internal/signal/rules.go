package signal

import (
	"sort"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// Both leaves decide resolution; a Declared zone input carries no version (ADR-0086, ADR-0024).

var leafVersions = sortedStrings(rw.Version, wd.Version)

func sortedStrings(s ...string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func All() []Rule {
	return []Rule{
		lameDelegation{},
		cnameTargetNameError{},
		zoneDeclaredNameReturnsNameError{},
		resolvedNameAbsentFromZone{},
		nonGloballyReachableFromInternet{},
	}
}

func RuleNames() []string { return AllRuleNames() }

func hasAnswer(outcome string) bool {
	return outcome == Resolved || outcome == Shadowed
}

type lameDelegation struct{}

func (lameDelegation) Name() string     { return "lame-delegation" }
func (lameDelegation) Version() Version { return Version{Rule: "v1", Composes: leafVersions} }

func (lameDelegation) Severity() Severity { return SevMedium }
func (lameDelegation) Eval(f NameFacts) Outcome {
	// A total domain is legal: nothing within the estate is excluded, only the withdrawn (ADR-0024).
	if !f.InEstate {
		return OutsideDomain
	}
	switch f.Resolution {
	case Lame:
		return Fired
	case Shadowed, Gap:
		return NotEvaluable
	default:
		return NotFired
	}
}

// A dangling alias to a name that does not exist is the classic sub-domain-takeover setup.

type cnameTargetNameError struct{}

func (cnameTargetNameError) Name() string     { return "cname-target-name-error" }
func (cnameTargetNameError) Version() Version { return Version{Rule: "v1", Composes: leafVersions} }

func (cnameTargetNameError) Severity() Severity { return SevHigh }
func (cnameTargetNameError) Eval(f NameFacts) Outcome {
	if f.CNAMETarget == "" {
		return OutsideDomain
	}
	if f.Resolution == Shadowed {
		return NotEvaluable
	}
	switch f.TargetResolution {
	case NameError:
		return Fired
	case "", Gap, Shadowed:
		return NotEvaluable
	default:
		return NotFired
	}
}

type zoneDeclaredNameReturnsNameError struct{}

func (zoneDeclaredNameReturnsNameError) Name() string {
	return "zone-declared-name-returns-name-error"
}
func (zoneDeclaredNameReturnsNameError) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}

// A declared name resolution cannot find is a zone hygiene gap, not a live exposure.

func (zoneDeclaredNameReturnsNameError) Severity() Severity { return SevMedium }
func (zoneDeclaredNameReturnsNameError) Eval(f NameFacts) Outcome {
	if !f.ZoneDeclared {
		return OutsideDomain
	}
	switch f.Resolution {
	case NameError:
		return Fired
	// A lame delegation makes the NameError unobtainable, so the rule cannot decide (ADR-0024, #128).
	case Lame, Shadowed, Gap:
		return NotEvaluable
	default:
		return NotFired
	}
}

type resolvedNameAbsentFromZone struct{}

func (resolvedNameAbsentFromZone) Name() string { return "resolved-name-absent-from-zone" }
func (resolvedNameAbsentFromZone) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}

func (resolvedNameAbsentFromZone) Severity() Severity { return SevLow }
func (resolvedNameAbsentFromZone) Eval(f NameFacts) Outcome {
	if !f.InDeclaredZone {
		return OutsideDomain
	}
	switch f.Resolution {
	case Resolved:
		if f.ZoneDeclared {
			return NotFired
		}
		return Fired
	case Shadowed:
		return NotEvaluable
	default:
		// The domain is names our resolver resolved, so an unresolved one is outside and not not-fired.
		return OutsideDomain
	}
}

type nonGloballyReachableFromInternet struct{}

func (nonGloballyReachableFromInternet) Name() string {
	return "non-globally-reachable-address-resolved-from-internet"
}
func (nonGloballyReachableFromInternet) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}

func (nonGloballyReachableFromInternet) Severity() Severity { return SevMedium }
func (nonGloballyReachableFromInternet) Eval(f NameFacts) Outcome {
	// Not assertable without an internet vantage, and the internal twin is a refused rule (ADR-0071).
	if !f.HasInternetVantage || !hasAnswer(f.InternetResolution) {
		return OutsideDomain
	}
	if f.InternetResolution == Shadowed {
		return NotEvaluable
	}
	if anyNonGloballyReachable(f.InternetAddresses) {
		return Fired
	}
	return NotFired
}
