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

func TestComposeShadowedOverwritesNameError(t *testing.T) {
	// The wildcard verdict wins the recorded value: a name resolution-walk read as
	// NameError composes to Shadowed. Both outcomes suppress the Name's membership
	// (WithdrawnCrossClass), so the composed value records what discriminated it —
	// the wildcard — rather than a bare NameError.
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

// A Name read Shadowed at every available vantage is suppressed — Shadowed
// decides membership as affirmatively as NameError (AC #192): the fictional name
// leaves the estate and cites no Address, while the discriminated real name and
// its address stay.
func TestMembershipShadowedNameSuppressed(t *testing.T) {
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

// Suppression is existential like presence: a Name Shadowed at one vantage but
// still Resolved at another is admitted by that vantage and stays present.
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

func TestMembershipSeedCoveredNameAlwaysPresent(t *testing.T) {
	ne := Compose(OutcomeNameError, nil, wd.VerdictNotShadowed)
	est := Membership([]Observation{
		{Name: "iana.org", Vantage: "v1", Resolution: ne},
	}, []string{"iana.org"}, nil)
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
