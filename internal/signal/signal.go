// Package signal is the `Signal` rule engine: a named, versioned rule evaluated
// over the current Derived state of the estate (CONTEXT.md `Signal`, v1 spec
// §5.2, ADR-0024). A rule is four parts — its `Predicate domain`, its predicate,
// its `not-evaluable` case, and its version vector — and nothing else. This
// package holds the engine and the five `Name`-only rules that need no port tier
// (`lame-delegation`, `cname-target-name-error`,
// `zone-declared-name-returns-name-error`, `resolved-name-absent-from-zone`,
// `non-globally-reachable-address-resolved-from-internet`).
//
// The engine is a pure function over an in-memory snapshot of Derived facts
// (NameFacts), decoupled from how those facts are stored. That is the deep-module
// seam: the web layer reads resolution / dns-record / membership out of the
// corpus and assembles the snapshot; the engine decides fired / did-not-fire /
// `not-evaluable` and never touches a database. It is what makes every rule's
// predicate, `not-evaluable` case and version testable hermetically.
package signal

import "sort"

// Outcome is a rule's verdict for one subject. The census has exactly three
// members — Fired, NotFired and NotEvaluable — over one population, the rule's
// `Predicate domain` (ADR-0024). OutsideDomain is deliberately NOT a census
// member: a subject outside the domain is not rendered at all — not a member,
// not a row, not a state, not a transition — because the rule's question does
// not arise for it. That is the line that keeps the `did-not-fire` column from
// swelling with subjects the rule was never about.
type Outcome string

const (
	// OutsideDomain: the subject is not in the rule's predicate domain. Not a
	// census member; the subject is not rendered by this rule at all.
	OutsideDomain Outcome = "outside-domain"
	// Fired: the predicate is true of the subject.
	Fired Outcome = "fired"
	// NotFired: the subject is in the domain and the predicate is false.
	NotFired Outcome = "not-fired"
	// NotEvaluable: the subject is in the domain but the rule cannot read the
	// answer — the evidence is a value about our own sight (`Shadowed`) or there
	// is no value at all (`Gap`, or evidence the rule declares that was never
	// measured). Distinct from NotFired: a rule excluded by fact or aperture is
	// not a clean bill of health (CONTEXT.md `Signal`).
	NotEvaluable Outcome = "not-evaluable"
)

// The composed `resolution` outcomes the five rules read, mirroring
// internal/estate's constants. `resolution-walk` decides Resolved / NoData /
// NameError / Gap; the dns-record NS delegation carries Lame; and
// `wildcard-discrimination` decides Shadowed. The engine reads the one composed
// value the web layer folds from those leaves.
const (
	Resolved  = "Resolved"
	NoData    = "NoData"
	NameError = "NameError"
	Lame      = "Lame"
	Shadowed  = "Shadowed"
	Gap       = "Gap"
)

// NameFacts is the current Derived state about one `Name` that the five
// Name-only rules read — the only evidence they declare, so the engine composes
// no leaf beyond what is folded into these fields (ADR-0024: a domain and
// predicate read one evidence set). Every field is a value about *now*: a census
// is current state and never a comparison.
type NameFacts struct {
	// Name is the subject key — the label sequence rendered per ADR-0055. It is
	// what every census member drills to (/subjects/{Name}).
	Name string

	// InEstate reports whether the Name is a current member of the estate — not
	// withdrawn (ADR-0006: a Name leaves only where our resolver reads NameError
	// across the vantage composition). It is the universe boundary for the one
	// **total**-domain rule (lame-delegation). A rule whose domain is a declared
	// input rather than the estate — the two zone rules, whose lifecycle is their
	// evidence's and not the subject's membership — ignores it and fires on a
	// withdrawn Name (CONTEXT.md `Signal`).
	InEstate bool

	// Resolution is the cross-class composed `resolution` outcome our resolver
	// reads for this Name: Resolved | NoData | NameError | Lame | Shadowed | Gap.
	// Lame folds the dns-record NS delegation's verdict; Shadowed folds
	// `wildcard-discrimination`. Addresses is the set a Resolved answer cites.
	Resolution string
	Addresses  []string

	// CNAMETarget is the target `Name` of this Name's dns-record CNAME, empty
	// when the Name holds no CNAME (which puts it outside cname-target-name-error's
	// domain). TargetResolution is that target's own composed resolution outcome,
	// empty when the target was never measured.
	CNAMETarget      string
	TargetResolution string

	// ZoneDeclared reports whether the operator's zone file declares this exact
	// Name. InDeclaredZone reports whether the Name falls within a name scope
	// that holds a zone file at all (label-wise suffix coverage).
	ZoneDeclared   bool
	InDeclaredZone bool

	// The internet-class view, for the one vantage-scoped rule
	// (`non-globally-reachable-address-resolved-from-internet`, ADR-0071). A
	// vantage-scoped claim is read only at the vantage that scopes it: with no
	// internet-class vantage the Name is outside that rule's domain entirely.
	HasInternetVantage bool
	InternetResolution string
	InternetAddresses  []string
}

// Version is a rule's version vector (ADR-0008/ADR-0024). Rule is the rule's own
// semantic version, moved on an output-affecting change to its domain, predicate
// or not-evaluable case. Composes lists the Derivation leaves the rule reads:
// where a rule is evaluated over other Derived values, its version composes
// theirs (CONTEXT.md `Signal`). Versions are per rule, never one set-wide
// version, so editing one rule leaves the rest comparable.
type Version struct {
	Rule     string
	Composes []string
}

// String renders the vector deterministically, so the golden gate can compare it
// byte for byte: the rule version, then each composed leaf version in sorted
// order, joined by "|".
func (v Version) String() string {
	out := "rule@" + v.Rule
	for _, c := range v.Composes {
		out += "|" + c
	}
	return out
}

// Rule is one `Signal`: a named, versioned rule that decides an Outcome for one
// subject's Derived facts. It has no lifecycle of its own and no severity — it is
// a named fact, and urgency belongs to the transition that surfaces it.
type Rule interface {
	// Name is the fact the rule reads, never a conclusion or a protocol
	// (CONTEXT.md `Signal`). Its extension is the rule's `Predicate domain`.
	Name() string
	// Version is the rule's version vector.
	Version() Version
	// Eval decides the subject's Outcome. Returning OutsideDomain excludes the
	// subject from the census entirely.
	Eval(f NameFacts) Outcome
}

// Member is one census member row: a subject key and nothing else. It is never
// the `Subjects` row component — it carries no `Citation` and rides no search,
// and a member list's header count is exactly its length (ADR-0102). Every
// member is drillable to its subject by this key.
type Member struct {
	Subject string
}

// Census is a rule's current-state census: three member lists over one
// population, the rule's `Predicate domain`. It is never a delta, trend or
// series — subtracting two censuses conflates a moved domain with a moved
// predicate (ADR-0024). Each member's count IS the length of its own list, so
// every member is enumerable in full and none is sampled, ranked, grouped or
// truncated — including a member that is most of the estate.
type Census struct {
	Rule         string
	Version      Version
	Fired        []Member
	NotFired     []Member
	NotEvaluable []Member
}

// InDomain is the size of the predicate domain — the sum of the three members,
// and the census's denominator. It never counts subjects outside the domain.
func (c Census) InDomain() int {
	return len(c.Fired) + len(c.NotFired) + len(c.NotEvaluable)
}

// Empty reports whether the predicate domain has no subject at all. An empty
// domain is legal and renders as a no-population panel, never as a census of
// zeroes (ADR-0024, CONTEXT.md `Predicate domain`).
func (c Census) Empty() bool { return c.InDomain() == 0 }

// Evaluate runs one rule over the current estate snapshot and buckets each
// subject into its census member — dropping the ones outside the domain, which
// are not rendered. Members are ordered by the subject and never by attention or
// the rule's verdict (ADR-0102), so the output is deterministic.
func Evaluate(r Rule, estate []NameFacts) Census {
	c := Census{Rule: r.Name(), Version: r.Version()}
	for _, f := range estate {
		switch r.Eval(f) {
		case Fired:
			c.Fired = append(c.Fired, Member{Subject: f.Name})
		case NotFired:
			c.NotFired = append(c.NotFired, Member{Subject: f.Name})
		case NotEvaluable:
			c.NotEvaluable = append(c.NotEvaluable, Member{Subject: f.Name})
		}
		// OutsideDomain falls through: not a census member.
	}
	sortMembers(c.Fired)
	sortMembers(c.NotFired)
	sortMembers(c.NotEvaluable)
	return c
}

// EvaluateAll runs every shipped rule over the estate, in a stable order.
func EvaluateAll(estate []NameFacts) []Census {
	rules := All()
	out := make([]Census, 0, len(rules))
	for _, r := range rules {
		out = append(out, Evaluate(r, estate))
	}
	return out
}

func sortMembers(m []Member) {
	sort.Slice(m, func(i, j int) bool { return m[i].Subject < m[j].Subject })
}
