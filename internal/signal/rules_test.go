package signal

import (
	"strings"
	"testing"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// ruleByName finds a shipped rule by its name for the per-rule tests.
func ruleByName(t *testing.T, name string) Rule {
	t.Helper()
	for _, r := range All() {
		if r.Name() == name {
			return r
		}
	}
	t.Fatalf("no rule named %q", name)
	return nil
}

func TestLameDelegation(t *testing.T) {
	r := ruleByName(t, "lame-delegation")
	cases := []struct {
		name string
		res  string
		want Outcome
	}{
		{"lame fires", Lame, Fired},
		{"resolved does not fire", Resolved, NotFired},
		{"nodata does not fire", NoData, NotFired},
		{"nameerror does not fire", NameError, NotFired},
		{"shadowed is not-evaluable, not not-fired", Shadowed, NotEvaluable},
		{"gap is not-evaluable", Gap, NotEvaluable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r.Eval(NameFacts{Name: "n", InEstate: true, Resolution: c.res})
			if got != c.want {
				t.Fatalf("resolution %s: got %s want %s", c.res, got, c.want)
			}
		})
	}

	// A withdrawn Name is not in the estate, so it is outside the total domain.
	if got := r.Eval(NameFacts{Name: "n", InEstate: false, Resolution: Lame}); got != OutsideDomain {
		t.Fatalf("withdrawn Name: got %s want OutsideDomain", got)
	}
}

func TestCnameTargetNameError(t *testing.T) {
	r := ruleByName(t, "cname-target-name-error")

	// A Name with no CNAME is outside the domain — not a census member.
	if got := r.Eval(NameFacts{Name: "n", Resolution: Resolved}); got != OutsideDomain {
		t.Fatalf("no CNAME: got %s want OutsideDomain", got)
	}
	// A dangling CNAME (target NameError) fires.
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "gone.example.com", TargetResolution: NameError}); got != Fired {
		t.Fatalf("dangling CNAME: got %s want Fired", got)
	}
	// A CNAME whose target resolves does not fire.
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "live.example.com", TargetResolution: Resolved}); got != NotFired {
		t.Fatalf("live target: got %s want NotFired", got)
	}
	// Shadowed on the name itself is not-evaluable, inside the domain.
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "x", Resolution: Shadowed, TargetResolution: NameError}); got != NotEvaluable {
		t.Fatalf("shadowed name: got %s want NotEvaluable", got)
	}
	// A target we never measured cannot decide the alias — not-evaluable, not a
	// clean not-fired.
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "unknown", TargetResolution: ""}); got != NotEvaluable {
		t.Fatalf("unmeasured target: got %s want NotEvaluable", got)
	}
}

func TestZoneDeclaredNameReturnsNameError(t *testing.T) {
	r := ruleByName(t, "zone-declared-name-returns-name-error")

	// A name the zone file does not declare is outside the domain.
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: false, Resolution: NameError}); got != OutsideDomain {
		t.Fatalf("not declared: got %s want OutsideDomain", got)
	}
	// A declared name our resolver NXDOMAINs fires.
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: NameError}); got != Fired {
		t.Fatalf("declared + NameError: got %s want Fired", got)
	}
	// A declared name that resolves does not fire.
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Resolved}); got != NotFired {
		t.Fatalf("declared + Resolved: got %s want NotFired", got)
	}
	// Lame makes the NameError unobtainable: not-evaluable, not fired.
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Lame}); got != NotEvaluable {
		t.Fatalf("declared + Lame: got %s want NotEvaluable", got)
	}
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("declared + Shadowed: got %s want NotEvaluable", got)
	}
}

func TestResolvedNameAbsentFromZone(t *testing.T) {
	r := ruleByName(t, "resolved-name-absent-from-zone")

	// Outside every declared zone: the question does not arise.
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: false, Resolution: Resolved}); got != OutsideDomain {
		t.Fatalf("outside zone: got %s want OutsideDomain", got)
	}
	// Resolved within a declared zone but absent from the zone file: fires.
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, ZoneDeclared: false, Resolution: Resolved}); got != Fired {
		t.Fatalf("resolved + absent: got %s want Fired", got)
	}
	// Resolved and declared: does not fire.
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, ZoneDeclared: true, Resolution: Resolved}); got != NotFired {
		t.Fatalf("resolved + declared: got %s want NotFired", got)
	}
	// Shadowed within a zone is not-evaluable, inside.
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, Resolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("shadowed: got %s want NotEvaluable", got)
	}
	// A name in the zone that did not resolve is not in the domain (rule is over
	// names our resolver *resolved*).
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, Resolution: NameError}); got != OutsideDomain {
		t.Fatalf("in zone + NameError: got %s want OutsideDomain", got)
	}
}

func TestNonGloballyReachableFromInternet(t *testing.T) {
	r := ruleByName(t, "non-globally-reachable-address-resolved-from-internet")

	// No internet-class vantage: outside the domain (vantage-scoped, ADR-0071).
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: false, InternetResolution: Resolved, InternetAddresses: []string{"10.0.0.1"}}); got != OutsideDomain {
		t.Fatalf("no internet vantage: got %s want OutsideDomain", got)
	}
	// Internet-class Resolved with a private address: fires.
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Resolved, InternetAddresses: []string{"93.184.216.34", "10.0.0.5"}}); got != Fired {
		t.Fatalf("private leak: got %s want Fired", got)
	}
	// Internet-class Resolved, all global: does not fire.
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Resolved, InternetAddresses: []string{"93.184.216.34"}}); got != NotFired {
		t.Fatalf("all global: got %s want NotFired", got)
	}
	// No answer (NameError): outside the domain — no address set to be about.
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: NameError}); got != OutsideDomain {
		t.Fatalf("NameError: got %s want OutsideDomain", got)
	}
	// Shadowed internet answer is not-evaluable, inside.
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("shadowed: got %s want NotEvaluable", got)
	}
}

// TestRuleVersions pins that every rule carries a version vector composing both
// leaves it reads — the version composes theirs (CONTEXT.md `Signal`). A leaf
// version bump moving these strings is the intended coupling.
func TestRuleVersions(t *testing.T) {
	for _, r := range All() {
		v := r.Version()
		if v.Rule == "" {
			t.Fatalf("%s: empty rule version", r.Name())
		}
		if len(v.Composes) != 2 {
			t.Fatalf("%s: composes %v, want the two resolution leaves", r.Name(), v.Composes)
		}
		joined := strings.Join(v.Composes, ",")
		if !strings.Contains(joined, rw.Version) || !strings.Contains(joined, wd.Version) {
			t.Fatalf("%s: version %q must compose %q and %q", r.Name(), v, rw.Version, wd.Version)
		}
		// Sorted and deterministic.
		if v.Composes[0] > v.Composes[1] {
			t.Fatalf("%s: composed versions not sorted: %v", r.Name(), v.Composes)
		}
	}
}

// TestRuleNamesStable pins the five names — a rule is named for the fact it
// reads and its name fixes its domain, so a rename is a domain change.
func TestRuleNamesStable(t *testing.T) {
	want := []string{
		"lame-delegation",
		"cname-target-name-error",
		"zone-declared-name-returns-name-error",
		"resolved-name-absent-from-zone",
		"non-globally-reachable-address-resolved-from-internet",
	}
	got := All()
	if len(got) != len(want) {
		t.Fatalf("got %d rules, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name() != name {
			t.Fatalf("rule %d: got %q want %q", i, got[i].Name(), name)
		}
	}
}
