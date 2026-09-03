package estate

import (
	"testing"

	"github.com/winniel123/verge-asm/internal/drift"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

func TestAddressPresentIsDisjunctive(t *testing.T) {
	// AC #195: presence is exactly a current citation or a Seed cover, the limbs disjunctive.
	cases := []struct {
		cited, seedCovered, want bool
	}{
		{false, false, false},
		{true, false, true},
		{false, true, true},
		{true, true, true},
	}
	for _, c := range cases {
		if got := AddressPresent(c.cited, c.seedCovered); got != c.want {
			t.Errorf("AddressPresent(cited=%v, seedCovered=%v) = %v, want %v",
				c.cited, c.seedCovered, got, c.want)
		}
	}
}

func TestAddressClosureUncitedWhenResolutionStopsCiting(t *testing.T) {
	// AC #195: evidence about another subject grounds the departure when a resolution stops citing.
	reason, left := AddressClosure(false, false, false)
	if !left {
		t.Fatal("an address neither cited nor Seed-covered must leave the estate")
	}
	if reason != drift.ReasonUncited {
		t.Errorf("closure ground = %q, want %q (a resolution stopped citing it)", reason, drift.ReasonUncited)
	}
}

func TestAddressClosureStaysWhileEitherLimbHolds(t *testing.T) {
	if _, left := AddressClosure(true, false, false); left {
		t.Error("a cited address must not leave")
	}
	if _, left := AddressClosure(false, true, false); left {
		t.Error("a Seed-covered address must not leave")
	}
	if _, left := AddressClosure(true, false, true); left {
		t.Error("a Seed narrowing must not withdraw an address a resolution still cites")
	}
}

func TestAddressClosureDescopedWhenSeedNarrows(t *testing.T) {
	reason, left := AddressClosure(false, false, true)
	if !left {
		t.Fatal("an address left by a Seed narrowing and cited by nothing must leave")
	}
	if reason != drift.ReasonDescoped {
		t.Errorf("closure ground = %q, want %q (our aperture stopped covering it)", reason, drift.ReasonDescoped)
	}
}

func TestMembershipSeedCoveredAddressPresentWithoutCitation(t *testing.T) {
	resolved := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	est := Membership(
		[]Observation{{Name: "real.example.com", Vantage: "v1", Resolution: resolved}},
		nil,
		[]string{"203.0.113.7"},
	)
	if !contains(est.Addresses, "203.0.113.7") {
		t.Error("a Seed-covered address must be in the estate before anything resolves to it")
	}
	if !contains(est.Addresses, "198.51.100.9") {
		t.Error("a cited address must remain in the estate (the limbs union)")
	}
}

func TestMembershipUncitedAddressLeaves(t *testing.T) {
	noData := Compose(OutcomeNoData, nil, wd.VerdictNotShadowed)
	est := Membership(
		[]Observation{{Name: "was.example.com", Vantage: "v1", Resolution: noData}},
		nil,
		nil,
	)
	if contains(est.Addresses, "203.0.113.7") {
		t.Error("an address no current resolution cites and no Seed covers must leave the estate")
	}
}
