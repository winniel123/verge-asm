package signal

// A rule reads exactly one subject kind, so the corpus carries three populations (ADR-0024).

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

func SubjectKindFor(ruleName string) string {
	// A Service or Endpoint key carries a /, so a drill-down rides ?key= not a path segment (#248).
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

func AllRuleNames() []string {
	// The guard admits exactly these, so an operator never accepts a firing that cannot occur (ADR-0187).
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
