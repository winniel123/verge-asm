package signal

import "testing"

// stubRule lets the engine tests drive Evaluate over a fixed outcome map without
// depending on a real rule's predicate.
type stubRule struct {
	name string
	out  map[string]Outcome
}

func (s stubRule) Name() string     { return s.name }
func (s stubRule) Version() Version { return Version{Rule: "vtest", Composes: []string{"a", "b"}} }
func (s stubRule) Eval(f NameFacts) Outcome {
	if o, ok := s.out[f.Name]; ok {
		return o
	}
	return OutsideDomain
}

func TestEvaluateBucketsAndDropsOutsideDomain(t *testing.T) {
	r := stubRule{name: "stub", out: map[string]Outcome{
		"b.example.com": Fired,
		"a.example.com": Fired,
		"c.example.com": NotFired,
		"d.example.com": NotEvaluable,
		"e.example.com": OutsideDomain,
	}}
	estate := []NameFacts{
		{Name: "e.example.com"}, {Name: "c.example.com"}, {Name: "b.example.com"},
		{Name: "d.example.com"}, {Name: "a.example.com"},
	}
	c := Evaluate(r, estate)

	// OutsideDomain is not rendered anywhere: the denominator excludes it.
	if c.InDomain() != 4 {
		t.Fatalf("InDomain = %d, want 4 (outside-domain excluded)", c.InDomain())
	}
	// Each member count IS its list length.
	if len(c.Fired) != 2 || len(c.NotFired) != 1 || len(c.NotEvaluable) != 1 {
		t.Fatalf("members: fired=%d notfired=%d ne=%d", len(c.Fired), len(c.NotFired), len(c.NotEvaluable))
	}
	// Members are ordered by subject, never by attention.
	if c.Fired[0].Subject != "a.example.com" || c.Fired[1].Subject != "b.example.com" {
		t.Fatalf("fired not ordered by subject: %v", c.Fired)
	}
	if c.Empty() {
		t.Fatalf("census with members reported Empty")
	}
}

func TestEmptyDomainIsNoPopulation(t *testing.T) {
	r := stubRule{name: "stub", out: map[string]Outcome{}} // everything OutsideDomain
	c := Evaluate(r, []NameFacts{{Name: "a"}, {Name: "b"}})
	if !c.Empty() {
		t.Fatalf("all-outside-domain census should be Empty (no-population panel), got InDomain=%d", c.InDomain())
	}
	if len(c.Fired)+len(c.NotFired)+len(c.NotEvaluable) != 0 {
		t.Fatalf("empty domain must render no members, not a census of zeroes")
	}
}

func TestVersionStringDeterministic(t *testing.T) {
	v := Version{Rule: "v1", Composes: []string{"resolution-walk/v1", "wildcard-discrimination/v1"}}
	const want = "rule@v1|resolution-walk/v1|wildcard-discrimination/v1"
	if v.String() != want {
		t.Fatalf("Version.String() = %q, want %q", v.String(), want)
	}
}

func TestEvaluateAllReturnsEveryRule(t *testing.T) {
	got := EvaluateAll([]NameFacts{{Name: "a", Resolution: Resolved}})
	if len(got) != len(All()) {
		t.Fatalf("EvaluateAll returned %d censuses, want %d", len(got), len(All()))
	}
	for i, c := range got {
		if c.Rule != All()[i].Name() {
			t.Fatalf("census %d rule = %q, want %q", i, c.Rule, All()[i].Name())
		}
	}
}
