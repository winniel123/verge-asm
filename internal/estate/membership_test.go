package estate

import (
	"reflect"
	"testing"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

func TestComposeShadowedCitesNothing(t *testing.T) {
	// The golden-corpus.md §8.2 citation pin, read from the membership side.
	got := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictShadowed)
	if got.Outcome != OutcomeShadowed || len(got.Addresses) != 0 {
		t.Errorf("Shadowed compose = %+v, want Shadowed citing nothing", got)
	}
}

func TestComposeNotShadowedCites(t *testing.T) {
	got := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	if got.Outcome != OutcomeResolved || !reflect.DeepEqual(got.Addresses, []string{"198.51.100.9"}) {
		t.Errorf("not-Shadowed compose = %+v, want Resolved citing the set", got)
	}
}

func TestComposeShadowedOverwritesNameError(t *testing.T) {
	// The recorded value names what discriminated it rather than a bare NameError.
	got := Compose(OutcomeNameError, nil, wd.VerdictShadowed)
	if got.Outcome != OutcomeShadowed {
		t.Errorf("NameError under Shadowed = %+v, want Shadowed", got)
	}
}

func TestComposeIncompleteProbeIsGap(t *testing.T) {
	got := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictGap)
	if got.Outcome != OutcomeGap || len(got.Addresses) != 0 {
		t.Errorf("Gap compose = %+v, want Gap citing nothing", got)
	}
}

func TestMembershipShadowedNameSuppressed(t *testing.T) {
	// AC #192: Shadowed decides membership as affirmatively as NameError.
	shadowed := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictShadowed)
	resolved := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "ghost.example.com", Vantage: "v1", Resolution: shadowed},
		{Name: "real.example.com", Vantage: "v1", Resolution: resolved},
	}, nil, nil)

	if contains(est.Names, "ghost.example.com") {
		t.Error("a Name Shadowed at every vantage is suppressed, not present")
	}
	if contains(est.Addresses, "203.0.113.1") {
		t.Error("a Shadowed answer cites no Address — 203.0.113.1 must leave the estate")
	}
	if !contains(est.Names, "real.example.com") || !contains(est.Addresses, "198.51.100.9") {
		t.Error("the discriminated real Name and its cited address stay in the estate")
	}
}

func TestMembershipShadowedSurvivesWhereAnotherVantageAdmits(t *testing.T) {
	shadowed := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictShadowed)
	resolved := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "split.example.com", Vantage: "v1", Resolution: shadowed},
		{Name: "split.example.com", Vantage: "v2", Resolution: resolved},
	}, nil, nil)

	if !contains(est.Names, "split.example.com") {
		t.Error("a Name still Resolved at some vantage is present despite a Shadowed vantage")
	}
	if !contains(est.Addresses, "198.51.100.9") {
		t.Error("the admitting vantage's cited address is in the estate")
	}
}

func TestMembershipNameErrorEverywhereWithdraws(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "dead.example.com", Vantage: "v1", Resolution: ne},
		{Name: "dead.example.com", Vantage: "v2", Resolution: ne},
	}, nil, nil)
	if contains(est.Names, "dead.example.com") {
		t.Error("a Name reading NameError from every vantage withdraws")
	}
}

func TestMembershipPresenceIsExistentialAcrossVantages(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	ok := Compose(OutcomeResolved, []string{"203.0.113.5"}, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "split.example.com", Vantage: "v1", Resolution: ne},
		{Name: "split.example.com", Vantage: "v2", Resolution: ok},
	}, nil, nil)
	if !contains(est.Names, "split.example.com") {
		t.Error("a Name resolving at some vantage is present (existential within witnesses)")
	}
	if !contains(est.Addresses, "203.0.113.5") {
		t.Error("the resolving vantage's address is cited")
	}
}

func TestMembershipSeedCoveredNameLeavesOnDecidedAbsence(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "iana.org", Vantage: "v1", Resolution: ne},
	}, []string{"iana.org"}, nil)
	if contains(est.Names, "iana.org") {
		t.Error("a Seed admits a Name, it does not outrank a decided cross-class Name Error")
	}
}

func TestMembershipSeedCoveredNameHeldThroughShadowed(t *testing.T) {
	shadowed := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictShadowed)
	est := Membership([]Observation{
		{Name: "ghost.example.com", Vantage: "v1", Resolution: shadowed},
	}, []string{"ghost.example.com"}, nil)
	if !contains(est.Names, "ghost.example.com") {
		t.Error("a Seed holds a Name through Shadowed — the residue stays visibly unconfirmed")
	}
}

func TestMembershipSeedCoveredNameAdmittedUnmeasured(t *testing.T) {
	est := Membership(nil, []string{"new.example.com"}, nil)
	if !contains(est.Names, "new.example.com") {
		t.Error("a declared Name is a subject before anything resolves")
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
