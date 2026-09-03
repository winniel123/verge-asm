package estate

// The dependency runs one way: drift imports nothing of ours, so estate may import it.

import "github.com/winniel123/verge-asm/internal/drift"

func AddressPresent(cited, seedCovered bool) bool {
	// An Address has no existence to observe, so presence is citation or Seed cover (ADR-0047).
	return cited || seedCovered
}

func AddressClosure(cited, seedCovered, seedDescoped bool) (reason drift.ClosureReason, left bool) {
	// The ground is decided here; drift applies the departure and decides none (ADR-0087).
	if AddressPresent(cited, seedCovered) {
		return "", false
	}
	if seedDescoped {
		// The operator's narrowing is the mover, and this ground alone blocks a spurious returned.
		return drift.ReasonDescoped, true
	}
	// The departure rests on evidence about the Name, never an observation of the address itself.
	return drift.ReasonUncited, true
}
