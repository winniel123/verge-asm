package signal

// Corpus is the full in-memory Derived snapshot the seventeen-rule set reads: the
// `Name` facts the five Name-only rules evaluate, the `Service` facts the two
// Service rules evaluate, and the `Endpoint` facts the ten Endpoint rules
// evaluate. The engine is split by subject kind — a rule reads exactly one kind —
// so the corpus carries three populations, and each is folded from the observation
// corpus by the web layer (ADR-0024). It is decoupled from how those facts are
// stored: the engine never touches a database.
type Corpus struct {
	Names     []NameFacts
	Services  []ServiceFacts
	Endpoints []EndpointFacts
}

func EvaluateCorpus(c Corpus) []Census {
	out := make([]Census, 0, len(All())+len(AllEndpointRules())+len(AllServiceRules()))
	for _, r := range All() {
		out = append(out, Evaluate(r, c.Names))
	}
	for _, r := range AllEndpointRules() {
		out = append(out, EvaluateEndpoint(r, c.Endpoints))
	}
	for _, r := range AllServiceRules() {
		out = append(out, EvaluateService(r, c.Services))
	}
	return out
}

// SubjectKindFor reports the subject kind — "name", "service" or "endpoint" —
// the named rule censuses, or "" for a rule that does not exist. The engine is
// split by subject kind (a rule reads exactly one), so a rule name determines the
// kind of every subject in its census and of every `Annotation` an operator
// declares against it. The Signals web layer reads this to build a drill-down
// link the subject's route actually serves — a Service or Endpoint key carries a
// `/`, so it must ride the `?key=` page, not a path segment (#248).
func SubjectKindFor(ruleName string) string {
	for _, r := range All() {
		if r.Name() == ruleName {
			return "name"
		}
	}
	for _, r := range AllEndpointRules() {
		if r.Name() == ruleName {
			return "endpoint"
		}
	}
	for _, r := range AllServiceRules() {
		if r.Name() == ruleName {
			return "service"
		}
	}
	return ""
}

// AllRuleNames returns the names of all seventeen shipped rules in EvaluateCorpus
// order — the set an `Annotation` may name, so the Signals declare form offers
// exactly the rules that exist across all three subject kinds and nothing an
// operator could accept a firing on that never fires.
func AllRuleNames() []string {
	out := make([]string, 0, 17)
	for _, r := range All() {
		out = append(out, r.Name())
	}
	for _, r := range AllEndpointRules() {
		out = append(out, r.Name())
	}
	for _, r := range AllServiceRules() {
		out = append(out, r.Name())
	}
	return out
}
