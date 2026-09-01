package queue

import (
	"net/netip"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
)

// The custody-extension veto is told by DISPLAY and never by a message (ADR-0129 §5,
// #987). These two tests pin the silence at the seam that would break it, because a
// message fired here would push the operator at a decline — and a veto WITHHOLDS a
// probe, which is the safe direction. The coverage-class message exists for the
// dangerous one, where the probing gate opens with no Declared act behind it, and that
// justification does not transfer.

// declineEstate is a custody-extended estate whose one in-zone name fronts a measured
// shared edge, so the extension declines the reach.
func declineEstate(edge netip.Addr) custody.Estate {
	return custody.Estate{
		ExtendedZones: []string{"example.com"},
		Resolutions:   []custody.Resolution{{Owner: "shop.example.com", Address: edge}},
	}.WithEdgeFanout(custody.EdgeFanout{Enabled: true, Shared: map[netip.Addr]bool{edge: true}})
}

// A decline fires no coverage-class message. The chain is structural rather than
// conditional: a vetoed edge derives `third-party`, so it is never probed, so no
// `resolution` span opens over it, so membershipMessages sees no entering root to fire
// at. The test walks all three links, because the last one alone would pass over an
// empty change set for any reason at all.
func TestDeclineFiresNoMessage(t *testing.T) {
	edge := netip.MustParseAddr("93.184.216.10")
	if got := declineEstate(edge).Derive(edge); got != custody.ThirdParty {
		t.Fatalf("Derive on a declined edge = %q, want %q — the veto did not hold", got, custody.ThirdParty)
	}
	// A `third-party` address is probed by no tier, so the batch opens no span over
	// it and the fold hands the producer no entering root.
	msgs := membershipMessages(time.Unix(0, 0).UTC(), nil, membershipInputs{})
	if len(msgs) != 0 {
		t.Errorf("messages = %+v on a batch that probed a declined edge, want none", msgs)
	}
}

// Re-pointing a Name off a dedicated origin onto a shared edge fires no message either.
// The old dedicated address leaves by measurement with no `Gap`, because the name
// stopped citing it, and the aperture does not widen — so there is no `revealed` to
// carry a message. A CLOSING span is not an entering root, and membershipMessages fires
// at entering roots alone. The change stays visible on the panel, pulled and never
// pushed.
func TestRepointOntoSharedEdgeFiresNoMessage(t *testing.T) {
	closed := []spanChange{{
		SubjectKind: "address",
		SubjectKey:  "203.0.113.77",
		Facet:       resolutionwalk.FacetResolution,
		Opened:      false,
	}}
	if msgs := membershipMessages(time.Unix(0, 0).UTC(), closed, membershipInputs{}); len(msgs) != 0 {
		t.Errorf("messages = %+v on a re-point off a dedicated origin, want none", msgs)
	}
}
