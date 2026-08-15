package estate

import (
	"testing"

	"github.com/winniel123/verge-asm/internal/drift"
	wd "github.com/winniel123/verge-asm/internal/measure/wildcarddiscrim"
)

// AC #195: an Address is in the estate exactly while a current resolution cites
// it OR a Seed covers it — the two limbs are disjunctive.
func TestAddressPresentIsDisjunctive(t *testing.T) {
	cases := []struct {
		cited, seedCovered, want bool
	}{
		{false, false, false}, // neither limb: absent
		{true, false, true},   // cited only
		{false, true, true},   // Seed-covered only (present before anything resolves)
		{true, true, true},    // both
	}
	for _, c := range cases {
		if got := AddressPresent(c.cited, c.seedCovered); got != c.want {
			t.Errorf("AddressPresent(cited=%v, seedCovered=%v) = %v, want %v",
				c.cited, c.seedCovered, got, c.want)
		}
	}
}

// AC #195: `uncited` Closure applies when a resolution stops citing an address —
// evidence about another subject grounds the departure.
func TestAddressClosureUncitedWhenResolutionStopsCiting(t *testing.T) {
	reason, left := AddressClosure(false /*cited*/, false /*seedCovered*/, false /*seedDescoped*/)
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
	// A Seed narrowing does not withdraw an address a resolution still cites.
	if _, left := AddressClosure(true, false, true); left {
		t.Error("a Seed narrowing must not withdraw an address a resolution still cites")
	}
}

func TestAddressClosureDescopedWhenSeedNarrows(t *testing.T) {
	reason, left := AddressClosure(false, false, true /*seedDescoped*/)
	if !left {
		t.Fatal("an address left by a Seed narrowing and cited by nothing must leave")
	}
	if reason != drift.ReasonDescoped {
		t.Errorf("closure ground = %q, want %q (our aperture stopped covering it)", reason, drift.ReasonDescoped)
	}
}

// Membership folds the Seed-covered Address limb into the Address estate: a
// Seed-covered address is present even where no current resolution cites it, and
// the two limbs union rather than replace.
func TestMembershipSeedCoveredAddressPresentWithoutCitation(t *testing.T) {
	resolved := Compose(OutcomeResolved, []string{"198.51.100.9"}, wd.VerdictNotShadowed)
	est := Membership(
		[]Observation{{Name: "real.example.com", Vantage: "v1", Resolution: resolved}},
		nil,
		[]string{"203.0.113.7"}, // an address-scope Seed enumerates this, nothing resolves to it
	)
	if !contains(est.Addresses, "203.0.113.7") {
		t.Error("a Seed-covered address must be in the estate before anything resolves to it")
	}
	if !contains(est.Addresses, "198.51.100.9") {
		t.Error("a cited address must remain in the estate (the limbs union)")
	}
}

// An address held only by a resolution that stopped citing it, with no Seed
// cover, is absent from the Address estate — the citation limb alone withdrew it.
func TestMembershipUncitedAddressLeaves(t *testing.T) {
	// The current resolution no longer cites 203.0.113.7 (it now cites nothing new).
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
