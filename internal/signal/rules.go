package signal

import (
	"sort"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// leafVersions is the sorted set of Derivation leaf versions every Name-only
// rule composes. All five read the `resolution` facet, which `resolution-walk`
// and `wildcard-discrimination` decide jointly (ADR-0086), so each rule's
// version vector composes both leaves' versions — a bump of either leaf moves
// every rule that reads the value it decides. The zone declaration these rules
// also read is Declared (input, carrying no version, ADR-0024), so it adds
// nothing to the vector.
var leafVersions = sortedStrings(rw.Version, wd.Version)

func sortedStrings(s ...string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// All returns the five shipped Name-only rules in a stable order: the order they
// render on the Signals page and the order the golden gate walks them.
func All() []Rule {
	return []Rule{
		lameDelegation{},
		cnameTargetNameError{},
		zoneDeclaredNameReturnsNameError{},
		resolvedNameAbsentFromZone{},
		nonGloballyReachableFromInternet{},
	}
}

// hasAnswer reports whether a composed resolution outcome carries an address set
// the world affirmed — the two answer-bearing values. NameError / NoData / Lame
// / Gap carry no answer.
func hasAnswer(outcome string) bool {
	return outcome == Resolved || outcome == Shadowed
}

// --- lame-delegation ------------------------------------------------------
//
// Domain: every `Name` in the estate — a **total** domain is legal (ADR-0024).
// Predicate: our delegation walk found the Name's authorities all reachable and
// all refusing — `Lame`. `not-evaluable`: `Shadowed` (a value about our own
// sight) and `Gap` (no delegation answer at all). Everything else — Resolved,
// NoData, NameError — is a determinate not-fired.

type lameDelegation struct{}

func (lameDelegation) Name() string     { return "lame-delegation" }
func (lameDelegation) Version() Version { return Version{Rule: "v1", Composes: leafVersions} }
func (lameDelegation) Eval(f NameFacts) Outcome {
	// The domain is total over the estate: a withdrawn Name is not in the estate,
	// so it is not in this rule's population — nothing *within* the estate is
	// excluded (ADR-0024: "a total domain is legal").
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

// --- cname-target-name-error ----------------------------------------------
//
// Domain: `Name`s whose `dns-record` holds a CNAME (ADR-0024). A Name with no
// CNAME is outside the domain — the question does not arise. Predicate: the
// CNAME target returns `NameError` — a dangling alias pointing at a name that
// does not exist, the classic sub-domain-takeover setup. `not-evaluable`:
// `Shadowed` on the name itself; and a target whose own resolution we could not
// read (never measured, `Gap`, or `Shadowed`), since the predicate needs the
// target's determinate outcome.

type cnameTargetNameError struct{}

func (cnameTargetNameError) Name() string     { return "cname-target-name-error" }
func (cnameTargetNameError) Version() Version { return Version{Rule: "v1", Composes: leafVersions} }
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
		// The target's own answer is unreadable, so the alias's health cannot be
		// decided — not a clean not-fired.
		return NotEvaluable
	default:
		return NotFired
	}
}

// --- zone-declared-name-returns-name-error --------------------------------
//
// Domain: `Name`s the operator's zone file declares (ADR-0024). A name the file
// does not declare is outside the domain. Predicate: our resolver returns
// `NameError` for a name the operator says should exist. `not-evaluable`:
// `Lame` — a lame delegation makes the `NameError` unobtainable (ADR-0024's #128
// amendment), so we cannot say whether the name would have returned one — and
// `Shadowed`, and `Gap`.

type zoneDeclaredNameReturnsNameError struct{}

func (zoneDeclaredNameReturnsNameError) Name() string {
	return "zone-declared-name-returns-name-error"
}
func (zoneDeclaredNameReturnsNameError) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}
func (zoneDeclaredNameReturnsNameError) Eval(f NameFacts) Outcome {
	if !f.ZoneDeclared {
		return OutsideDomain
	}
	switch f.Resolution {
	case NameError:
		return Fired
	case Lame, Shadowed, Gap:
		return NotEvaluable
	default:
		return NotFired
	}
}

// --- resolved-name-absent-from-zone ---------------------------------------
//
// Domain: `Name`s our resolver resolved **within a declared zone** (ADR-0024) —
// a name that both falls inside a name scope holding a zone file and carries an
// answer. A name outside every declared zone, or one with no answer, is outside
// the domain: the question (is a name resolving inside your zone that your zone
// file does not declare?) does not arise. Predicate: the name is absent from the
// zone file. `not-evaluable`: `Shadowed` — we cannot confirm the answer is real,
// so we cannot say the name genuinely resolves.

type resolvedNameAbsentFromZone struct{}

func (resolvedNameAbsentFromZone) Name() string { return "resolved-name-absent-from-zone" }
func (resolvedNameAbsentFromZone) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}
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
		// NameError / NoData / Lame / Gap: our resolver did not resolve it, so it
		// is not in the domain (Names our resolver *resolved* within a zone).
		return OutsideDomain
	}
}

// --- non-globally-reachable-address-resolved-from-internet ----------------
//
// Domain: `Name`s whose **internet-class** `resolution` holds an answer —
// `Resolved` or `Shadowed` (ADR-0024/#128). NameError, NoData and Lame carry no
// address set, so the question does not arise. The domain carries a
// `Vantage class` (ADR-0071): the fact — *a non-globally-reachable address was
// resolved from the internet* — is not assertable of a name nobody asked about
// from the internet, so with no internet-class vantage the name is outside the
// domain, and the internal twin is a different (refused) rule. Predicate: some
// address in the internet-class answer is not globally reachable — an internal
// address leaking into a public DNS answer. `not-evaluable`: `Shadowed` (our own
// sight).

type nonGloballyReachableFromInternet struct{}

func (nonGloballyReachableFromInternet) Name() string {
	return "non-globally-reachable-address-resolved-from-internet"
}
func (nonGloballyReachableFromInternet) Version() Version {
	return Version{Rule: "v1", Composes: leafVersions}
}
func (nonGloballyReachableFromInternet) Eval(f NameFacts) Outcome {
	if !f.HasInternetVantage || !hasAnswer(f.InternetResolution) {
		return OutsideDomain
	}
	if f.InternetResolution == Shadowed {
		return NotEvaluable
	}
	// Resolved: fired where any cited address is non-globally-reachable.
	if anyNonGloballyReachable(f.InternetAddresses) {
		return Fired
	}
	return NotFired
}
