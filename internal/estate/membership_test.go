package estate

import (
	"reflect"
	"testing"

	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// Compose is the site where both membership leaves meet. These cases are the
// citation pin (golden-corpus.md §8.2) read from the membership side.
func TestComposeShadowedCitesNothing(t *testing.T) {
	// resolution-walk measured Resolved with an address set; wildcard-discrimination
	// says Shadowed. The composed value cites nothing — the address set leaves.
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

func TestComposeShadowedSuppressesNameErrorWithdrawal(t *testing.T) {
	// The W6 case: a name resolution-walk read as NameError (a withdrawal) is
	// overwritten by Shadowed, so it does not withdraw.
	got := Compose(OutcomeNameError, nil, wd.VerdictShadowed)
	if got.Outcome != OutcomeShadowed {
		t.Errorf("NameError under Shadowed = %+v, want Shadowed (no withdrawal)", got)
	}
}

func TestComposeIncompleteProbeIsGap(t *testing.T) {
	got := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictGap)
	if got.Outcome != OutcomeGap || len(got.Addresses) != 0 {
		t.Errorf("Gap compose = %+v, want Gap citing nothing", got)
	}
}

// Membership reads the composed value, so a Shadowed Name is suppressed: it stays
// present (not withdrawn) yet cites no Address (AC #192).
func TestMembershipShadowedNamePresentButUncited(t *testing.T) {
	shadowed := Compose(OutcomeResolved, []string{"203.0.113.1"}, wd.VerdictShadowed)
	resolved := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "ghost.example.com", Vantage: "v1", Resolution: shadowed},
		{Name: "real.example.com", Vantage: "v1", Resolution: resolved},
	}, nil)

	if !contains(est.Names, "ghost.example.com") {
		t.Error("a Shadowed Name must stay present (it does not withdraw)")
	}
	if contains(est.Addresses, "203.0.113.1") {
		t.Error("a Shadowed Name cites no Address — 203.0.113.1 must leave the estate")
	}
	if !contains(est.Addresses, "198.51.100.9") {
		t.Error("the discriminated Name's cited address must be in the estate")
	}
}

func TestMembershipNameErrorEverywhereWithdraws(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "dead.example.com", Vantage: "v1", Resolution: ne},
		{Name: "dead.example.com", Vantage: "v2", Resolution: ne},
	}, nil)
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
	}, nil)
	if !contains(est.Names, "split.example.com") {
		t.Error("a Name resolving at some vantage is present (existential within witnesses)")
	}
	if !contains(est.Addresses, "203.0.113.5") {
		t.Error("the resolving vantage's address is cited")
	}
}

func TestMembershipSeedCoveredNameAlwaysPresent(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "iana.org", Vantage: "v1", Resolution: ne},
	}, []string{"iana.org"})
	if !contains(est.Names, "iana.org") {
		t.Error("a Seed-covered Name is Declared and stays present")
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
