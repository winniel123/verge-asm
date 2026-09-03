// Package signal evaluates named, versioned rules over the estate's current Derived
// state (CONTEXT.md `Signal`, v1 spec §5.2, ADR-0024). It is a pure function over an
// in-memory snapshot, decoupled from storage: the engine never touches a database.
package signal

import "sort"

// Rendered nowhere, so the did-not-fire column never swells with subjects the rule was never about.

type Outcome string

const (
	// A rule's domain is where its fact is assertable at all, so nothing outside it is a member.

	OutsideDomain Outcome = "outside-domain"
	Fired         Outcome = "fired"
	NotFired      Outcome = "not-fired"

	// Excluded by fact or aperture is not a clean bill of health (CONTEXT.md `Signal`).

	NotEvaluable Outcome = "not-evaluable"
)

// These mirror internal/estate's constants: the engine reads the folded value, never a leaf.

const (
	Resolved  = "Resolved"
	NoData    = "NoData"
	NameError = "NameError"
	Lame      = "Lame"
	Shadowed  = "Shadowed"
	Gap       = "Gap"
)

// A rule reads exactly the evidence it declares (ADR-0024), so no leaf is composed beyond these.

type NameFacts struct {
	Name string

	// A rule whose domain is a declared input ignores this and fires on a withdrawn Name.

	InEstate bool

	Resolution string
	Addresses  []string

	CNAMETarget      string
	TargetResolution string

	ZoneDeclared   bool
	InDeclaredZone bool

	HasInternetVantage bool
	InternetResolution string
	InternetAddresses  []string
}

// Per rule and never set-wide, so editing one rule leaves every other rule's censuses comparable.

type Version struct {
	Rule     string
	Composes []string
}

func (v Version) String() string {
	// Tests byte-compare this rendering, so the caller sorts Composes and this never reorders.
	out := "rule@" + v.Rule
	for _, c := range v.Composes {
		out += "|" + c
	}
	return out
}

// A rule is named for the fact it reads, never a conclusion or a protocol (CONTEXT.md `Signal`).

type Rule interface {
	Name() string
	Version() Version
	Severity() Severity
	Eval(f NameFacts) Outcome
}

// Never the Subjects row component: it carries no Citation and rides no search (ADR-0102).

type Member struct {
	Subject string
}

// Subtracting two conflates a moved domain with a moved predicate, so never a delta (ADR-0024).

type Census struct {
	Rule         string
	Version      Version
	Fired        []Member
	NotFired     []Member
	NotEvaluable []Member
}

func (c Census) InDomain() int {
	// Every member is enumerable in full: none is sampled, ranked, grouped or truncated (ADR-0102).
	return len(c.Fired) + len(c.NotFired) + len(c.NotEvaluable)
}

// An empty domain is legal and renders a no-population panel, never a census of zeroes (ADR-0024).

func (c Census) Empty() bool { return c.InDomain() == 0 }

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
	}
	// Ordered by subject and never by attention or verdict, so the output is deterministic (ADR-0102).
	sortMembers(c.Fired)
	sortMembers(c.NotFired)
	sortMembers(c.NotEvaluable)
	return c
}

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
