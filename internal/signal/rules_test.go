package signal

import (
	"strings"
	"testing"

	rw "github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

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

	if got := r.Eval(NameFacts{Name: "n", InEstate: false, Resolution: Lame}); got != OutsideDomain {
		t.Fatalf("withdrawn Name: got %s want OutsideDomain", got)
	}
}

func TestCnameTargetNameError(t *testing.T) {
	r := ruleByName(t, "cname-target-name-error")

	if got := r.Eval(NameFacts{Name: "n", Resolution: Resolved}); got != OutsideDomain {
		t.Fatalf("no CNAME: got %s want OutsideDomain", got)
	}
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "gone.example.com", TargetResolution: NameError}); got != Fired {
		t.Fatalf("dangling CNAME: got %s want Fired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "live.example.com", TargetResolution: Resolved}); got != NotFired {
		t.Fatalf("live target: got %s want NotFired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "x", Resolution: Shadowed, TargetResolution: NameError}); got != NotEvaluable {
		t.Fatalf("shadowed name: got %s want NotEvaluable", got)
	}
	if got := r.Eval(NameFacts{Name: "n", CNAMETarget: "unknown", TargetResolution: ""}); got != NotEvaluable {
		t.Fatalf("unmeasured target: got %s want NotEvaluable", got)
	}
}

func TestZoneDeclaredNameReturnsNameError(t *testing.T) {
	r := ruleByName(t, "zone-declared-name-returns-name-error")

	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: false, Resolution: NameError}); got != OutsideDomain {
		t.Fatalf("not declared: got %s want OutsideDomain", got)
	}
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: NameError}); got != Fired {
		t.Fatalf("declared + NameError: got %s want Fired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Resolved}); got != NotFired {
		t.Fatalf("declared + Resolved: got %s want NotFired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Lame}); got != NotEvaluable {
		t.Fatalf("declared + Lame: got %s want NotEvaluable", got)
	}
	if got := r.Eval(NameFacts{Name: "n", ZoneDeclared: true, Resolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("declared + Shadowed: got %s want NotEvaluable", got)
	}
}

func TestResolvedNameAbsentFromZone(t *testing.T) {
	r := ruleByName(t, "resolved-name-absent-from-zone")

	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: false, Resolution: Resolved}); got != OutsideDomain {
		t.Fatalf("outside zone: got %s want OutsideDomain", got)
	}
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, ZoneDeclared: false, Resolution: Resolved}); got != Fired {
		t.Fatalf("resolved + absent: got %s want Fired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, ZoneDeclared: true, Resolution: Resolved}); got != NotFired {
		t.Fatalf("resolved + declared: got %s want NotFired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, Resolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("shadowed: got %s want NotEvaluable", got)
	}
	if got := r.Eval(NameFacts{Name: "n", InDeclaredZone: true, Resolution: NameError}); got != OutsideDomain {
		t.Fatalf("in zone + NameError: got %s want OutsideDomain", got)
	}
}

func TestNonGloballyReachableFromInternet(t *testing.T) {
	r := ruleByName(t, "non-globally-reachable-address-resolved-from-internet")

	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: false, InternetResolution: Resolved, InternetAddresses: []string{"10.0.0.1"}}); got != OutsideDomain {
		t.Fatalf("no internet vantage: got %s want OutsideDomain", got)
	}
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Resolved, InternetAddresses: []string{"93.184.216.34", "10.0.0.5"}}); got != Fired {
		t.Fatalf("private leak: got %s want Fired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Resolved, InternetAddresses: []string{"93.184.216.34"}}); got != NotFired {
		t.Fatalf("all global: got %s want NotFired", got)
	}
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: NameError}); got != OutsideDomain {
		t.Fatalf("NameError: got %s want OutsideDomain", got)
	}
	if got := r.Eval(NameFacts{Name: "n", HasInternetVantage: true, InternetResolution: Shadowed}); got != NotEvaluable {
		t.Fatalf("shadowed: got %s want NotEvaluable", got)
	}
}

func TestRuleVersions(t *testing.T) {
	// A leaf version bump moving these strings is the intended coupling (CONTEXT.md `Signal`).
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
		if v.Composes[0] > v.Composes[1] {
			t.Fatalf("%s: composed versions not sorted: %v", r.Name(), v.Composes)
		}
	}
}

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
